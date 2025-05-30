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

// ProposeCheckStatus checks if there is an identical active proposal already.
// If not, it calls Propose(...) to create a new one.
func ProposeCheckStatus(
	checkType, checkName, memberName, domainName, endpoint string,
	status bool,
	errorText string,
	dataMap map[string]interface{},
) bool {

	State.Mu.RLock()
	for _, pt := range State.Proposals {
		prop := pt.Proposal
		if !pt.Finalized &&
			prop.CheckType == checkType &&
			prop.CheckName == checkName &&
			prop.MemberName == memberName &&
			prop.DomainName == domainName &&
			prop.Endpoint == endpoint &&
			prop.ProposedStatus == status {
			State.Mu.RUnlock()
			log.Log(log.Debug,
				"[NATS] ProposeCheckStatus skipped; identical active proposal for checkType=%s checkName=%s member=%s",
				checkType, checkName, memberName)
			return false
		}
	}
	State.Mu.RUnlock()

	log.Log(log.Debug,
		"[NATS] ProposeCheckStatus creating new proposal for checkType=%s checkName=%s member=%s domain=%s endpoint=%s status=%v",
		checkType, checkName, memberName, domainName, endpoint, status)

	Propose(checkType, checkName, memberName, domainName, endpoint, status, errorText, dataMap)
	return true
}

// Propose creates a new proposal, stores it, and publishes it to 'consensus.propose'.
func Propose(
	checkType, checkName, memberName, domainName, endpoint string,
	proposedStatus bool,
	errorText string,
	data map[string]interface{},
) (ProposalID, error) {

	pid := ProposalID(uuid.New().String())
	prop := Proposal{
		ID:             pid,
		CheckType:      checkType,
		CheckName:      checkName,
		MemberName:     memberName,
		DomainName:     domainName,
		Endpoint:       endpoint,
		ProposedStatus: proposedStatus,
		ErrorText:      errorText,
		Data:           data,
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
		"[NATS] Propose: Stored new proposal ID=%s, checkType=%s, checkName=%s, domain=%s, endpoint=%s, status=%v",
		pid, checkType, checkName, domainName, endpoint, proposedStatus)

	pt.Timer = time.AfterFunc(State.ProposalTimeout, func() {
		finalizeVote(pid)
	})

	dataBytes, _ := json.Marshal(prop)
	err := Publish(State.SubjectPropose, dataBytes) // using connection.go’s Publish
	if err != nil {
		log.Log(log.Error, "[NATS] Propose: failed to publish proposal ID=%s: %v", pid, err)
	} else {
		log.Log(log.Debug, "[NATS] Propose: published proposal ID=%s -> subject=%s", pid, State.SubjectPropose)
	}
	return pid, err
}

// handleProposal processes incoming "consensus.propose" messages.
func handleProposal(m *nats.Msg) {
	var prop Proposal
	if err := json.Unmarshal(m.Data, &prop); err != nil {
		log.Log(log.Error, "[NATS] handleProposal: unmarshal error: %v", err)
		return
	}

	log.Log(log.Debug,
		"[NATS] handleProposal: got proposal ID=%s, checkType=%s, checkName=%s, domain=%s, endpoint=%s, status=%v",
		prop.ID, prop.CheckType, prop.CheckName, prop.DomainName, prop.Endpoint, prop.ProposedStatus)

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

	// Immediately vote based on local data
	go func(pr Proposal) {
		found, localStatus := checkLocalStatus(pr.CheckType, pr.CheckName, pr.MemberName, pr.DomainName, pr.Endpoint)
		if !found {
			log.Log(log.Debug,
				"[NATS] handleProposal: local check not found for proposal ID=%s", pr.ID)
			return
		}
		v := Vote{
			ProposalID: pr.ID,
			NodeID:     State.NodeID,
			Agree:      (localStatus == pr.ProposedStatus),
			Timestamp:  time.Now().UTC(),
		}
		data, _ := json.Marshal(v)
		_ = Publish(State.SubjectVote, data) // using connection.go’s Publish

		log.Log(log.Debug,
			"[NATS] handleProposal: node=%s voted (agree=%v) for proposal ID=%s",
			State.NodeID, v.Agree, pr.ID)

	}(prop)
}

// handleVote processes incoming "consensus.vote" messages.
func handleVote(m *nats.Msg) {
	var vote Vote
	if err := json.Unmarshal(m.Data, &vote); err != nil {
		log.Log(log.Error, "[NATS] handleVote: unmarshal error: %v", err)
		return
	}

	log.Log(log.Debug,
		"[NATS] handleVote: got vote from node=%s for proposal ID=%s (agree=%v)",
		vote.NodeID, vote.ProposalID, vote.Agree)

	State.Mu.Lock()
	pt, exists := State.Proposals[vote.ProposalID]
	if !exists {
		log.Log(log.Debug, "[NATS] handleVote: no such proposal ID=%s, ignoring vote", vote.ProposalID)
		State.Mu.Unlock()
		return
	}
	if pt.Finalized {
		log.Log(log.Debug, "[NATS] handleVote: proposal ID=%s is already finalized, ignoring extra vote", vote.ProposalID)
		State.Mu.Unlock()
		return
	}

	pt.Votes[vote.NodeID] = vote.Agree

	monitorCount := countNodesByRole("IBPMonitor")
	if monitorCount == 0 {
		log.Log(log.Warn,
			"[NATS] handleVote: monitorCount=0, cannot finalize proposal ID=%s", vote.ProposalID)
		State.Mu.Unlock()
		return
	}
	majority := (monitorCount / 2) + 1

	yesCount := 0
	noCount := 0
	for nodeID, v := range pt.Votes {
		node, inMap := State.ClusterNodes[nodeID]
		if inMap && node.NodeRole == "IBPMonitor" {
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
		log.Log(log.Debug, "[NATS] handleVote: finalizing proposal ID=%s => accepted", vote.ProposalID)
		finalizeVote(vote.ProposalID)
		return

	} else if noCount >= majority && !pt.Finalized {
		pt.Finalized = true
		State.Mu.Unlock()
		log.Log(log.Debug, "[NATS] handleVote: finalizing proposal ID=%s => rejected", vote.ProposalID)
		finalizeVote(vote.ProposalID)
		return
	}

	// If all monitors have voted, finalize as well
	if len(pt.Votes) == monitorCount && !pt.Finalized {
		pt.Finalized = true
		State.Mu.Unlock()
		log.Log(log.Debug, "[NATS] handleVote: finalizing proposal ID=%s => all monitors voted", vote.ProposalID)
		finalizeVote(vote.ProposalID)
		return
	}

	State.Mu.Unlock()
}

// handleFinalize processes incoming "consensus.finalize" messages.
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
			"[NATS] handleFinalize: proposal ID=%s not found or already finalized", fm.ProposalID)
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

	monitorCount := countNodesByRole("IBPMonitor")
	if monitorCount == 0 {
		log.Log(log.Warn,
			"[NATS] finalizeVote: monitorCount=0 for proposal ID=%s, cannot determine majority", pid)
		pt.Finalized = true
		State.Mu.Unlock()
		return
	}

	yesCount := 0
	noCount := 0
	for nodeID, v := range pt.Votes {
		node, inMap := State.ClusterNodes[nodeID]
		if inMap && node.NodeRole == "IBPMonitor" {
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
		"[NATS] finalizeVote: proposal ID=%s => finalStatus=%v (yesCount=%d noCount=%d monitors=%d majority=%d)",
		pid, finalStatus, yesCount, noCount, monitorCount, majority)

	// Apply final status
	go applyOfficialChanges(pt.Proposal, finalStatus)

	// Publish a "consensus.finalize" broadcast
	fm := FinalizeMessage{
		ProposalID:  pid,
		FinalStatus: finalStatus,
		DecidedAt:   time.Now().UTC(),
	}
	data, _ := json.Marshal(fm)
	_ = Publish(State.SubjectFinalize, data) // just use connection.go’s Publish

	log.Log(log.Info,
		"[NATS] finalizeVote: proposal ID=%s => final status=%v", pid, finalStatus)
}

// checkLocalStatus consults local data about site/domain/endpoint
func checkLocalStatus(
	checkType, checkName, memberName, domainName, endpoint string,
) (bool, bool) {

	switch checkType {
	case "site":
		found, status := dat.GetLocalSiteStatus(checkName, memberName)
		return found, status
	case "domain":
		found, status := dat.GetLocalDomainStatus(checkName, memberName, domainName)
		return found, status
	case "endpoint":
		found, status := dat.GetLocalEndpointStatus(checkName, memberName, endpoint)
		return found, status
	default:
		return false, false
	}
}

// applyOfficialChanges updates data.Official with up/down status
func applyOfficialChanges(prop Proposal, final bool) {
	chk, chkOk := findCheckByName(prop.CheckName, prop.CheckType)
	if !chkOk {
		log.Log(log.Warn, "[NATS] applyOfficialChanges: no check named=%s (type=%s)",
			prop.CheckName, prop.CheckType)
		return
	}
	mem, memOk := findMemberByName(prop.MemberName)
	if !memOk {
		log.Log(log.Warn, "[NATS] applyOfficialChanges: no member named=%s", prop.MemberName)
		return
	}

	var svc cfg.Service
	if prop.CheckType == "domain" || prop.CheckType == "endpoint" {
		serviceObj, ok := findServiceForDomain(prop.DomainName)
		if !ok && prop.CheckType == "domain" {
			log.Log(log.Warn,
				"[NATS] applyOfficialChanges: domain service not found for domain=%s",
				prop.DomainName)
			return
		}
		svc = serviceObj
	}

	status := final
	errorMsg := prop.ErrorText
	dataMap := prop.Data

	switch prop.CheckType {
	case "site":
		log.Log(log.Info,
			"[NATS] applyOfficialChanges: final site check for member=%s => %t",
			prop.MemberName, status)
		dat.UpdateOfficialSiteResult(chk, mem, status, errorMsg, dataMap)

	case "domain":
		log.Log(log.Info,
			"[NATS] applyOfficialChanges: final domain check for member=%s => %t domain=%s",
			prop.MemberName, status, prop.DomainName)
		dat.UpdateOfficialDomainResult(chk, mem, svc, prop.DomainName, status, errorMsg, dataMap)

	case "endpoint":
		log.Log(log.Info,
			"[NATS] applyOfficialChanges: final endpoint check for member=%s => %t domain=%s endpoint=%s",
			prop.MemberName, status, prop.DomainName, prop.Endpoint)
		dat.UpdateOfficialEndpointResult(chk, mem, svc, prop.DomainName, prop.Endpoint,
			status, errorMsg, dataMap)

	default:
		log.Log(log.Warn, "[NATS] applyOfficialChanges: unrecognized checkType=%s", prop.CheckType)
	}
}
