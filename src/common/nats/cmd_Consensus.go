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

const minConsensusVotes = 2 // hard floor per spec

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

func handleVote(m *nats.Msg) {
	var vote Vote
	if err := json.Unmarshal(m.Data, &vote); err != nil {
		log.Log(log.Error, "[NATS] handleVote: unmarshal error: %v", err)
		return
	}

	markNodeHeard(vote.SenderNodeID)

	State.Mu.Lock()
	defer State.Mu.Unlock()

	pt, exists := State.Proposals[vote.ProposalID]
	if !exists || pt.Finalized {
		return
	}
	pt.Votes[vote.NodeID] = vote.Agree

	monitorCount := countActiveMonitorsLocked()
	if monitorCount == 0 {
		return
	}
	majority := (monitorCount / 2) + 1

	yes, no := 0, 0
	for nid, v := range pt.Votes {
		if node, ok := State.ClusterNodes[nid]; ok && node.NodeRole == "IBPMonitor" && isNodeActive(node) {
			if v {
				yes++
			} else {
				no++
			}
		}
	}

	// finalise only if both the majority rule AND min‑votes rule are satisfied
	if yes >= majority && yes >= minConsensusVotes {
		pt.Finalized = true
		pt.FinalStatus = true
	} else if no >= majority && no >= minConsensusVotes {
		pt.Finalized = true
		pt.FinalStatus = false
	}

	if pt.Finalized {
		if pt.Timer != nil {
			pt.Timer.Stop()
		}
		go finalizeVote(vote.ProposalID) // run outside lock
	}
}

/* ───────────────────────── finalizeVote ──────────────────────────────── */

func finalizeVote(pid ProposalID) {
	State.Mu.Lock()
	pt, ok := State.Proposals[pid]
	if !ok {
		State.Mu.Unlock()
		return
	}

	// recount with latest liveness
	monitorCount := countActiveMonitorsLocked()
	yes, no := 0, 0
	for nid, v := range pt.Votes {
		if node, ok := State.ClusterNodes[nid]; ok && node.NodeRole == "IBPMonitor" && isNodeActive(node) {
			if v {
				yes++
			} else {
				no++
			}
		}
	}
	majority := (monitorCount / 2) + 1

	// if still not enough votes, extend timer and wait
	if yes < minConsensusVotes && no < minConsensusVotes {
		if pt.Timer != nil {
			pt.Timer.Reset(State.ProposalTimeout)
		} else {
			pt.Timer = time.AfterFunc(State.ProposalTimeout, func() { finalizeVote(pid) })
		}
		State.Mu.Unlock()
		return
	}

	// compute outcome respecting majority + minVotes
	if yes >= majority && yes >= minConsensusVotes {
		pt.FinalStatus = true
	} else if no >= majority && no >= minConsensusVotes {
		pt.FinalStatus = false
	} else {
		// tie or not enough – do not change official status
		State.Mu.Unlock()
		return
	}

	pt.Finalized = true
	State.Proposals[pid] = pt
	State.Mu.Unlock()

	// apply locally
	go applyOfficialChanges(pt.Proposal, pt.FinalStatus)

	// broadcast finalize (includes full proposal)
	fm := FinalizeMessage{
		ProposalID:  pid,
		Proposal:    pt.Proposal,
		FinalStatus: pt.FinalStatus,
		DecidedAt:   time.Now().UTC(),
	}
	data, _ := json.Marshal(fm)
	_ = Publish(State.SubjectFinalize, data)
}

/* ───────────────────────── handleFinalize ────────────────────────────── */

func handleFinalize(m *nats.Msg) {
	var fm FinalizeMessage
	if err := json.Unmarshal(m.Data, &fm); err != nil {
		log.Log(log.Error, "[NATS] handleFinalize: unmarshal error: %v", err)
		return
	}

	markNodeHeard(fm.Proposal.SenderNodeID)

	State.Mu.Lock()
	pt, exists := State.Proposals[fm.ProposalID]
	if !exists {
		pt = &ProposalTracking{
			Proposal: fm.Proposal,
			Votes:    make(map[string]bool),
		}
		State.Proposals[fm.ProposalID] = pt
	}
	pt.Finalized = true
	pt.FinalStatus = fm.FinalStatus
	if pt.Timer != nil {
		pt.Timer.Stop()
	}
	State.Mu.Unlock()

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
