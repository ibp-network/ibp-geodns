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
			prop.IsIPv6 == isIPv6 {
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

	// Start timer for forced finalize AFTER storing proposal
	pt.Timer = time.AfterFunc(State.ProposalTimeout, func() {
		log.Log(log.Debug, "[NATS] Proposal timeout reached for ID=%s, forcing finalization", pid)
		finalizeVote(pid)
	})
	State.Mu.Unlock()

	log.Log(log.Debug,
		"[NATS] Propose: Stored new proposal ID=%s, checkType=%s, checkName=%s, domain=%s, endpoint=%s, status=%v, isIPv6=%v from sender=%s",
		pid, checkType, checkName, domainName, endpoint, proposedStatus, isIPv6, State.NodeID)

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
			log.Log(log.Debug, "[NATS] Proposal timeout reached for ID=%s (from handleProposal), forcing finalization", prop.ID)
			finalizeVote(prop.ID)
		})
		log.Log(log.Debug, "[NATS] handleProposal: stored new proposal ID=%s", prop.ID)
		State.Mu.Unlock()

		// Vote immediately after storing
		go voteOnProposal(prop)
	} else {
		log.Log(log.Debug,
			"[NATS] handleProposal: proposal ID=%s already exists (finalized=%v)",
			existingPT.Proposal.ID, existingPT.Finalized)
		State.Mu.Unlock()
	}
}

// voteOnProposal is separated out to avoid holding locks
func voteOnProposal(proposal Proposal) {
	// Small delay to ensure the proposal is fully propagated
	time.Sleep(100 * time.Millisecond)

	found, localStatus := checkLocalStatus(proposal.CheckType, proposal.CheckName, proposal.MemberName, proposal.DomainName, proposal.Endpoint, proposal.IsIPv6)
	if !found {
		log.Log(log.Debug, "[NATS] voteOnProposal: local check not found for ID=%s", proposal.ID)
		return
	}

	v := Vote{
		ProposalID:   proposal.ID,
		SenderNodeID: State.NodeID,
		NodeID:       State.NodeID,
		Agree:        (localStatus == proposal.ProposedStatus),
		Timestamp:    time.Now().UTC(),
	}

	data, _ := json.Marshal(v)
	err := Publish(State.SubjectVote, data)
	if err != nil {
		log.Log(log.Error, "[NATS] voteOnProposal: failed to publish vote ID=%s: %v", proposal.ID, err)
	} else {
		log.Log(log.Debug, "[NATS] voteOnProposal: node=%s voted (agree=%v) for ID=%s", State.NodeID, v.Agree, proposal.ID)
	}
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
	defer State.Mu.Unlock()

	pt, exists := State.Proposals[vote.ProposalID]
	if !exists {
		log.Log(log.Debug, "[NATS] handleVote: no such proposal ID=%s, ignoring", vote.ProposalID)
		return
	}
	if pt.Finalized {
		log.Log(log.Debug, "[NATS] handleVote: proposal ID=%s is already finalized, ignoring extra vote", vote.ProposalID)
		return
	}

	pt.Votes[vote.NodeID] = vote.Agree

	// Count active monitors and votes
	monitorCount := countActiveMonitorsLocked() // Use locked version since we hold the lock
	if monitorCount == 0 {
		log.Log(log.Warn, "[NATS] handleVote: monitorCount=0, cannot finalize proposal=%s", vote.ProposalID)
		return
	}

	majority := (monitorCount / 2) + 1
	yesCount := 0
	noCount := 0

	// Count votes from active monitors
	for nodeID, v := range pt.Votes {
		if node, ok := State.ClusterNodes[nodeID]; ok && node.NodeRole == "IBPMonitor" && isNodeActive(node) {
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

	// Check if we should finalize
	shouldFinalize := false
	if yesCount >= majority {
		pt.Finalized = true
		shouldFinalize = true
		log.Log(log.Debug, "[NATS] handleVote: proposal ID=%s reached YES majority, finalizing", vote.ProposalID)
	} else if noCount >= majority {
		pt.Finalized = true
		shouldFinalize = true
		log.Log(log.Debug, "[NATS] handleVote: proposal ID=%s reached NO majority, finalizing", vote.ProposalID)
	} else if yesCount+noCount >= monitorCount {
		pt.Finalized = true
		shouldFinalize = true
		log.Log(log.Debug, "[NATS] handleVote: proposal ID=%s all monitors voted, finalizing", vote.ProposalID)
	}

	if shouldFinalize {
		// Cancel the timer if it exists
		if pt.Timer != nil {
			pt.Timer.Stop()
		}
		// Finalize in a goroutine to avoid holding lock
		go finalizeVote(vote.ProposalID)
	}
}

func finalizeVote(pid ProposalID) {
	State.Mu.Lock()
	pt, exists := State.Proposals[pid]
	if !exists {
		log.Log(log.Debug, "[NATS] finalizeVote: proposal ID=%s not found", pid)
		State.Mu.Unlock()
		return
	}

	if pt.Finalized {
		log.Log(log.Debug, "[NATS] finalizeVote: proposal ID=%s already finalized, skipping", pid)
		State.Mu.Unlock()
		return
	}

	if pt.Timer != nil {
		pt.Timer.Stop()
	}

	// vote counting (unchanged) …
	monitorCount := countActiveMonitorsLocked()
	yesCount := 0
	noCount := 0
	for nodeID, v := range pt.Votes {
		if node, ok := State.ClusterNodes[nodeID]; ok && node.NodeRole == "IBPMonitor" && isNodeActive(node) {
			if v {
				yesCount++
			} else {
				noCount++
			}
		}
	}
	majority := (monitorCount / 2) + 1

	var finalStatus bool
	if monitorCount == 0 {
		finalStatus = pt.Proposal.ProposedStatus
	} else if yesCount >= majority {
		finalStatus = true
	} else if noCount >= majority {
		finalStatus = false
	} else {
		finalStatus = (yesCount > noCount)
		if yesCount == noCount {
			finalStatus = pt.Proposal.ProposedStatus
		}
	}

	pt.FinalStatus = finalStatus
	pt.Finalized = true
	State.Proposals[pid] = pt // save
	State.Mu.Unlock()

	log.Log(log.Debug,
		"[NATS] finalizeVote: FINALIZED proposal ID=%s => finalStatus=%v (yes=%d no=%d monitors=%d majority=%d)",
		pid, finalStatus, yesCount, noCount, monitorCount, majority)

	// Apply locally
	go applyOfficialChanges(pt.Proposal, finalStatus)

	// Broadcast finalize (NOW WITH FULL PROPOSAL)
	fm := FinalizeMessage{
		ProposalID:  pid,
		Proposal:    pt.Proposal,
		FinalStatus: finalStatus,
		DecidedAt:   time.Now().UTC(),
	}
	data, _ := json.Marshal(fm)
	if err := Publish(State.SubjectFinalize, data); err != nil {
		log.Log(log.Error, "[NATS] finalizeVote: failed to publish finalize message: %v", err)
	}
}

// -----------------------------------------------------------------------------
// handleFinalize — now idempotent & tolerant of re‑ordering
// -----------------------------------------------------------------------------
func handleFinalize(m *nats.Msg) {
	var fm FinalizeMessage
	if err := json.Unmarshal(m.Data, &fm); err != nil {
		log.Log(log.Error, "[NATS] handleFinalize: unmarshal error: %v", err)
		return
	}

	// Mark node heard
	markNodeHeard(fm.Proposal.SenderNodeID)

	log.Log(log.Debug,
		"[NATS] handleFinalize: received finalize for ID=%s finalStatus=%v (check=%s/%s member=%s isIPv6=%v)",
		fm.ProposalID, fm.FinalStatus, fm.Proposal.CheckType, fm.Proposal.CheckName, fm.Proposal.MemberName, fm.Proposal.IsIPv6)

	// Ensure proposal tracking exists
	State.Mu.Lock()
	pt, exists := State.Proposals[fm.ProposalID]
	if !exists {
		// create minimal tracking so we don't lose history
		pt = &ProposalTracking{
			Proposal: fm.Proposal,
			Votes:    make(map[string]bool),
		}
		State.Proposals[fm.ProposalID] = pt
	}
	// mark finalized
	pt.Finalized = true
	pt.FinalStatus = fm.FinalStatus
	if pt.Timer != nil {
		pt.Timer.Stop()
	}
	State.Mu.Unlock()

	// Apply the change (idempotent)
	go applyOfficialChanges(fm.Proposal, fm.FinalStatus)
}

// countActiveMonitorsLocked counts active monitors while already holding the lock
func countActiveMonitorsLocked() int {
	n := 0
	for _, node := range State.ClusterNodes {
		if node.NodeRole == "IBPMonitor" && isNodeActive(node) {
			n++
		}
	}
	return n
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
			"[NATS] applyOfficialChanges: APPLYING site check for member=%s => %t isIPv6=%v",
			prop.MemberName, status, prop.IsIPv6)
		dat.UpdateOfficialSiteResult(chk, mem, status, errorMsg, dataMap, prop.IsIPv6)

	case "domain":
		log.Log(log.Debug,
			"[NATS] applyOfficialChanges: APPLYING domain check for member=%s => %t domain=%s isIPv6=%v",
			prop.MemberName, status, prop.DomainName, prop.IsIPv6)
		dat.UpdateOfficialDomainResult(chk, mem, svc, prop.DomainName, status, errorMsg, dataMap, prop.IsIPv6)

	case "endpoint":
		log.Log(log.Debug,
			"[NATS] applyOfficialChanges: APPLYING endpoint check for member=%s => %t domain=%s endpoint=%s isIPv6=%v",
			prop.MemberName, status, prop.DomainName, prop.Endpoint, prop.IsIPv6)
		dat.UpdateOfficialEndpointResult(chk, mem, svc, prop.DomainName, prop.Endpoint, status, errorMsg, dataMap, prop.IsIPv6)

	default:
		log.Log(log.Warn, "[NATS] applyOfficialChanges: unrecognized checkType=%s", prop.CheckType)
	}
}
