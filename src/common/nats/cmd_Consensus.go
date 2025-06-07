package nats

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
)

// ProposeCheckStatus is called after local checks detect status change.
func ProposeCheckStatus(
	checkType, checkName, memberName, domainName, endpoint string,
	status bool,
	errorText string,
	dataMap map[string]interface{},
	isIPv6 bool,
) bool {
	// Check for existing proposal without holding lock during Propose call
	shouldPropose := true

	State.Mu.RLock()
	for _, pt := range State.Proposals {
		prop := pt.Proposal
		// Only skip if it's an identical active proposal from *this node*.
		if !pt.Finalized &&
			prop.CheckType == checkType &&
			prop.CheckName == checkName &&
			prop.MemberName == memberName &&
			prop.DomainName == domainName &&
			prop.Endpoint == endpoint &&
			prop.ProposedStatus == status &&
			prop.IsIPv6 == isIPv6 &&
			prop.SenderNodeID == State.NodeID {
			log.Log(log.Debug,
				"[NATS] ProposeCheckStatus skipped; identical active proposal for checkType=%s checkName=%s member=%s isIPv6=%v from same node=%s",
				checkType, checkName, memberName, isIPv6, State.NodeID)
			shouldPropose = false
			break
		}
	}
	State.Mu.RUnlock()

	if shouldPropose {
		log.Log(log.Debug,
			"[NATS] ProposeCheckStatus creating new proposal for checkType=%s checkName=%s member=%s domain=%s endpoint=%s status=%v isIPv6=%v",
			checkType, checkName, memberName, domainName, endpoint, status, isIPv6)
		Propose(checkType, checkName, memberName, domainName, endpoint, status, errorText, dataMap, isIPv6)
		return true
	}
	return false
}

// Propose constructs a new Proposal and publishes it.
func Propose(
	checkType, checkName, memberName, domainName, endpoint string,
	proposedStatus bool,
	errorText string,
	data map[string]interface{},
	isIPv6 bool,
) (ProposalID, error) {
	pid := ProposalID(uuid.New().String())

	prop := Proposal{
		ID:             pid,
		SenderNodeID:   State.NodeID, // Track who sent it
		CheckType:      checkType,
		CheckName:      checkName,
		MemberName:     memberName,
		DomainName:     domainName,
		Endpoint:       endpoint,
		ProposedStatus: proposedStatus,
		ErrorText:      errorText,
		Data:           data,
		IsIPv6:         isIPv6,
		Timestamp:      time.Now().UTC(),
	}

	pt := &ProposalTracking{
		Proposal: prop,
		Votes:    make(map[string]bool),
	}

	State.Mu.Lock()
	State.Proposals[pid] = pt
	State.Mu.Unlock()

	log.Log(log.Debug,
		"[NATS] Propose: Stored new proposal ID=%s, checkType=%s, checkName=%s, domain=%s, endpoint=%s, status=%v, isIPv6=%v from sender=%s",
		pid, checkType, checkName, domainName, endpoint, proposedStatus, isIPv6, State.NodeID)

	// Start timer for forced finalize
	pt.Timer = time.AfterFunc(State.ProposalTimeout, func() {
		finalizeVote(pid)
	})

	dataBytes, _ := json.Marshal(prop)
	err := Publish(State.SubjectPropose, dataBytes)
	if err != nil {
		log.Log(log.Error, "[NATS] Propose: failed to publish ID=%s: %v", pid, err)
	} else {
		log.Log(log.Debug, "[NATS] Propose: published proposal ID=%s -> subject=%s", pid, State.SubjectPropose)
	}
	return pid, err
}

// handleProposal processes "consensus.propose"
func handleProposal(m *nats.Msg) {
	// Mark that we heard from this node. The 'sender' is in the JSON, but we might also want to parse from m.Reply, etc.
	// We'll do it once we decode to get prop.SenderNodeID
	var prop Proposal
	if err := json.Unmarshal(m.Data, &prop); err != nil {
		log.Log(log.Error, "[NATS] handleProposal: unmarshal error: %v", err)
		return
	}

	// Mark node as heard
	markNodeHeard(prop.SenderNodeID)

	log.Log(log.Debug,
		"[NATS] handleProposal: got proposal ID=%s, checkType=%s, checkName=%s, domain=%s, endpoint=%s, status=%v, isIPv6=%v from sender=%s",
		prop.ID, prop.CheckType, prop.CheckName, prop.DomainName, prop.Endpoint, prop.ProposedStatus, prop.IsIPv6, prop.SenderNodeID)

	State.Mu.Lock()
	existingPT, exists := State.Proposals[prop.ID]
	if !exists {
		pt := &ProposalTracking{
			Proposal: prop,
			Votes:    make(map[string]bool),
		}
		State.Proposals[prop.ID] = pt
		pt.Timer = time.AfterFunc(State.ProposalTimeout, func() {
			finalizeVote(prop.ID)
		})
		log.Log(log.Debug, "[NATS] handleProposal: stored new proposal ID=%s", prop.ID)
	} else {
		log.Log(log.Debug,
			"[NATS] handleProposal: proposal ID=%s already exists (finalized=%v)",
			existingPT.Proposal.ID, existingPT.Finalized)
	}
	State.Mu.Unlock()

	// After storing, we vote - moved outside of lock to prevent deadlock
	// Use a copy of the proposal to avoid race conditions
	proposalCopy := prop
	go func() {
		// Small delay to ensure the proposal is fully propagated
		time.Sleep(100 * time.Millisecond)

		found, localStatus := checkLocalStatus(proposalCopy.CheckType, proposalCopy.CheckName, proposalCopy.MemberName, proposalCopy.DomainName, proposalCopy.Endpoint, proposalCopy.IsIPv6)
		if !found {
			log.Log(log.Debug, "[NATS] handleProposal: local check not found for ID=%s", proposalCopy.ID)
			return
		}
		v := Vote{
			ProposalID:   proposalCopy.ID,
			SenderNodeID: State.NodeID,
			NodeID:       State.NodeID,
			Agree:        (localStatus == proposalCopy.ProposedStatus),
			Timestamp:    time.Now().UTC(),
		}
		data, _ := json.Marshal(v)
		err := Publish(State.SubjectVote, data)
		if err != nil {
			log.Log(log.Error, "[NATS] handleProposal: failed to publish vote ID=%s: %v", proposalCopy.ID, err)
		} else {
			log.Log(log.Debug, "[NATS] handleProposal: node=%s voted (agree=%v) for ID=%s", State.NodeID, v.Agree, proposalCopy.ID)
		}
	}()
}

// handleVote processes "consensus.vote"
func handleVote(m *nats.Msg) {
	var vote Vote
	if err := json.Unmarshal(m.Data, &vote); err != nil {
		log.Log(log.Error, "[NATS] handleVote: unmarshal error: %v", err)
		return
	}

	// Mark we heard from the voting node
	markNodeHeard(vote.SenderNodeID)

	log.Log(log.Debug,
		"[NATS] handleVote: got vote from node=%s for proposal ID=%s (agree=%v)",
		vote.NodeID, vote.ProposalID, vote.Agree)

	State.Mu.Lock()
	pt, exists := State.Proposals[vote.ProposalID]
	if !exists {
		log.Log(log.Debug, "[NATS] handleVote: no such proposal ID=%s, ignoring", vote.ProposalID)
		State.Mu.Unlock()
		return
	}
	if pt.Finalized {
		log.Log(log.Debug, "[NATS] handleVote: proposal ID=%s is finalized, ignoring extra vote", vote.ProposalID)
		State.Mu.Unlock()
		return
	}

	pt.Votes[vote.NodeID] = vote.Agree

	monitorCount := countActiveMonitors()
	if monitorCount == 0 {
		log.Log(log.Warn, "[NATS] handleVote: monitorCount=0, cannot finalize proposal=%s", vote.ProposalID)
		State.Mu.Unlock()
		return
	}
	majority := (monitorCount / 2) + 1

	yesCount := 0
	noCount := 0
	for nodeID, v := range pt.Votes {
		node, inMap := State.ClusterNodes[nodeID]
		if inMap && node.NodeRole == "IBPMonitor" && isNodeActive(node) {
			if v {
				yesCount++
			} else {
				noCount++
			}
		}
	}

	log.Log(log.Debug,
		"[NATS] handleVote: proposal ID=%s => yesCount=%d noCount=%d monitorCount=%d majority=%d",
		vote.ProposalID, yesCount, noCount, monitorCount, majority)

	if yesCount >= majority && !pt.Finalized {
		pt.Finalized = true
		State.Mu.Unlock()
		log.Log(log.Debug, "[NATS] handleVote: finalizing ID=%s => accepted", vote.ProposalID)
		finalizeVote(vote.ProposalID)
		return
	}
	if noCount >= majority && !pt.Finalized {
		pt.Finalized = true
		State.Mu.Unlock()
		log.Log(log.Debug, "[NATS] handleVote: finalizing ID=%s => rejected", vote.ProposalID)
		finalizeVote(vote.ProposalID)
		return
	}
	if len(pt.Votes) == monitorCount && !pt.Finalized {
		pt.Finalized = true
		State.Mu.Unlock()
		log.Log(log.Debug, "[NATS] handleVote: finalizing ID=%s => all monitors voted", vote.ProposalID)
		finalizeVote(vote.ProposalID)
		return
	}
	State.Mu.Unlock()
}

// handleFinalize processes "consensus.finalize"
func handleFinalize(m *nats.Msg) {
	var fm FinalizeMessage
	if err := json.Unmarshal(m.Data, &fm); err != nil {
		log.Log(log.Error, "[NATS] handleFinalize: unmarshal error: %v", err)
		return
	}

	log.Log(log.Debug,
		"[NATS] handleFinalize: got finalize for proposal ID=%s finalStatus=%v",
		fm.ProposalID, fm.FinalStatus)

	State.Mu.RLock()
	pt, exists := State.Proposals[fm.ProposalID]
	State.Mu.RUnlock()
	if exists && !pt.Finalized {
		State.Mu.Lock()
		pt.Finalized = true
		State.Mu.Unlock()
		go finalizeVote(pt.Proposal.ID)
	} else {
		log.Log(log.Debug,
			"[NATS] handleFinalize: proposal ID=%s not found or already finalized",
			fm.ProposalID)
	}
}

// finalizeVote is triggered by timeouts or forced finalization
func finalizeVote(pid ProposalID) {
	State.Mu.Lock()
	pt, exists := State.Proposals[pid]
	if !exists {
		log.Log(log.Debug, "[NATS] finalizeVote: proposal ID=%s not found", pid)
		State.Mu.Unlock()
		return
	}
	if pt.Timer != nil {
		pt.Timer.Stop()
	}

	monitorCount := countActiveMonitors()
	if monitorCount == 0 {
		log.Log(log.Warn, "[NATS] finalizeVote: monitorCount=0 for proposal=%s", pid)
		pt.Finalized = true
		State.Mu.Unlock()
		return
	}

	yesCount := 0
	noCount := 0
	for nodeID, v := range pt.Votes {
		node, inMap := State.ClusterNodes[nodeID]
		if inMap && node.NodeRole == "IBPMonitor" && isNodeActive(node) {
			if v {
				yesCount++
			} else {
				noCount++
			}
		}
	}
	majority := (monitorCount / 2) + 1

	var finalStatus bool
	switch {
	case yesCount >= majority:
		finalStatus = true
	case noCount >= majority:
		finalStatus = false
	default:
		finalStatus = (yesCount > noCount)
	}

	pt.FinalStatus = finalStatus
	pt.Finalized = true
	State.Mu.Unlock()

	log.Log(log.Debug,
		"[NATS] finalizeVote: proposal ID=%s => finalStatus=%v (yes=%d no=%d monitors=%d majority=%d)",
		pid, finalStatus, yesCount, noCount, monitorCount, majority)

	// apply to official data
	go applyOfficialChanges(pt.Proposal, finalStatus)

	// broadcast finalize
	fm := FinalizeMessage{
		ProposalID:  pid,
		FinalStatus: finalStatus,
		DecidedAt:   time.Now().UTC(),
	}
	data, _ := json.Marshal(fm)
	_ = Publish(State.SubjectFinalize, data)

	log.Log(log.Debug,
		"[NATS] finalizeVote: proposal ID=%s => final status=%v",
		pid, finalStatus)
}

// checkLocalStatus returns (found, status).
func checkLocalStatus(
	checkType, checkName, memberName, domainName, endpoint string,
	isIPv6 bool,
) (bool, bool) {
	switch checkType {
	case "site":
		found, status := dat.GetLocalSiteStatusIPv4v6(checkName, memberName, isIPv6)
		return found, status
	case "domain":
		found, status := dat.GetLocalDomainStatusIPv4v6(checkName, memberName, domainName, isIPv6)
		return found, status
	case "endpoint":
		found, status := dat.GetLocalEndpointStatusIPv4v6(checkName, memberName, domainName, endpoint, isIPv6)
		return found, status
	default:
		return false, false
	}
}

// applyOfficialChanges modifies data.Official after finalization
func applyOfficialChanges(prop Proposal, final bool) {
	chk, okChk := findCheckByName(prop.CheckName, prop.CheckType)
	if !okChk {
		log.Log(log.Warn, "[NATS] applyOfficialChanges: no check named=%s (type=%s)", prop.CheckName, prop.CheckType)
		return
	}
	mem, okMem := findMemberByName(prop.MemberName)
	if !okMem {
		log.Log(log.Warn, "[NATS] applyOfficialChanges: no member named=%s", prop.MemberName)
		return
	}

	var svc cfg.Service
	if prop.CheckType == "domain" || prop.CheckType == "endpoint" {
		serviceObj, ok := findServiceForDomain(prop.DomainName)
		if !ok && prop.CheckType == "domain" {
			log.Log(log.Warn, "[NATS] applyOfficialChanges: domain service not found for domain=%s", prop.DomainName)
			return
		}
		svc = serviceObj
	}

	status := final
	errorMsg := prop.ErrorText
	dataMap := prop.Data

	switch prop.CheckType {
	case "site":
		log.Log(log.Debug,
			"[NATS] applyOfficialChanges: final site check for member=%s => %t isIPv6=%v",
			prop.MemberName, status, prop.IsIPv6)
		dat.UpdateOfficialSiteResult(chk, mem, status, errorMsg, dataMap, prop.IsIPv6)

	case "domain":
		log.Log(log.Debug,
			"[NATS] applyOfficialChanges: final domain check for member=%s => %t domain=%s isIPv6=%v",
			prop.MemberName, status, prop.DomainName, prop.IsIPv6)
		dat.UpdateOfficialDomainResult(chk, mem, svc, prop.DomainName, status, errorMsg, dataMap, prop.IsIPv6)

	case "endpoint":
		log.Log(log.Debug,
			"[NATS] applyOfficialChanges: final endpoint check for member=%s => %t domain=%s endpoint=%s isIPv6=%v",
			prop.MemberName, status, prop.DomainName, prop.Endpoint, prop.IsIPv6)
		dat.UpdateOfficialEndpointResult(chk, mem, svc, prop.DomainName, prop.Endpoint, status, errorMsg, dataMap, prop.IsIPv6)

	default:
		log.Log(log.Warn, "[NATS] applyOfficialChanges: unrecognized checkType=%s", prop.CheckType)
	}
}
