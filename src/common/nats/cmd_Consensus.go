package nats

/*
   Consensus engine – production version.

   Key points
   ----------

   • Every local status divergence prompts ONE dedicated proposal.
   • Proposal outcome (Pass/Fail) is broadcast; when *Pass* all nodes
     apply exactly the **ProposedStatus** (never their own local copy).
   • Finalisation honours quorum ≥ ⌊N/2⌋+1 **and** a hard floor
     minConsensusVotes (2).
*/

import (
	"encoding/json"
	"time"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

/*─────────────────────────────────────────────────────────────
  CONSTANTS
─────────────────────────────────────────────────────────────*/

const minConsensusVotes = 2 // hard minimum even for small clusters

/*─────────────────────────────────────────────────────────────
  PUBLIC ENTRY – called by monitor helpers
─────────────────────────────────────────────────────────────*/

func ProposeCheckStatus(
	checkType, checkName, memberName,
	domainName, endpoint string,
	status bool,
	errorText string,
	dataMap map[string]interface{},
	isIPv6 bool,
) {
	// skip if identical active proposal already exists from this node
	State.Mu.RLock()
	for _, pt := range State.Proposals {
		if !pt.Finalized &&
			pt.Proposal.CheckType == checkType &&
			pt.Proposal.CheckName == checkName &&
			pt.Proposal.MemberName == memberName &&
			pt.Proposal.DomainName == domainName &&
			pt.Proposal.Endpoint == endpoint &&
			pt.Proposal.ProposedStatus == status &&
			pt.Proposal.IsIPv6 == isIPv6 {
			State.Mu.RUnlock()
			return
		}
	}
	State.Mu.RUnlock()

	propose(checkType, checkName, memberName, domainName, endpoint,
		status, errorText, dataMap, isIPv6)
}

/*─────────────────────────────────────────────────────────────
  CREATE + PUBLISH PROPOSAL
─────────────────────────────────────────────────────────────*/

func propose(
	checkType, checkName, memberName, domainName, endpoint string,
	status bool,
	errorText string,
	data map[string]interface{},
	isIPv6 bool,
) {
	pid := ProposalID(uuid.New().String())

	prop := Proposal{
		ID:             pid,
		SenderNodeID:   State.NodeID,
		CheckType:      checkType,
		CheckName:      checkName,
		MemberName:     memberName,
		DomainName:     domainName,
		Endpoint:       endpoint,
		ProposedStatus: status,
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
	if State.Proposals == nil {
		State.Proposals = make(map[ProposalID]*ProposalTracking)
	}
	State.Proposals[pid] = pt
	pt.Timer = time.AfterFunc(State.ProposalTimeout, func() { forceFinalize(pid) })
	State.Mu.Unlock()

	if dataBytes, _ := json.Marshal(prop); Publish(State.SubjectPropose, dataBytes) != nil {
		log.Log(log.Error, "[NATS] failed to publish proposal %s", pid)
	}

	go voteOnProposal(prop)
}

/*─────────────────────────────────────────────────────────────
  PROPOSAL HANDLER
─────────────────────────────────────────────────────────────*/

func handleProposal(m *nats.Msg) {
	var prop Proposal
	if err := json.Unmarshal(m.Data, &prop); err != nil {
		log.Log(log.Error, "[NATS] handleProposal: unmarshal error: %v", err)
		return
	}
	markNodeHeard(prop.SenderNodeID)

	State.Mu.Lock()
	if _, exists := State.Proposals[prop.ID]; !exists {
		State.Proposals[prop.ID] = &ProposalTracking{
			Proposal: prop,
			Votes:    make(map[string]bool),
		}
		State.Proposals[prop.ID].Timer = time.AfterFunc(State.ProposalTimeout, func() { forceFinalize(prop.ID) })
		State.Mu.Unlock()
		go voteOnProposal(prop)
		return
	}
	State.Mu.Unlock()
}

/*─────────────────────────────────────────────────────────────
  VOTING
─────────────────────────────────────────────────────────────*/

func voteOnProposal(prop Proposal) {
	time.Sleep(50 * time.Millisecond) // allow storage propagation

	found, localStatus := checkLocalStatus(
		prop.CheckType, prop.CheckName, prop.MemberName,
		prop.DomainName, prop.Endpoint, prop.IsIPv6)
	if !found {
		return
	}

	v := Vote{
		ProposalID:   prop.ID,
		SenderNodeID: State.NodeID,
		NodeID:       State.NodeID,
		Agree:        localStatus == prop.ProposedStatus,
		Timestamp:    time.Now().UTC(),
	}

	if data, _ := json.Marshal(v); Publish(State.SubjectVote, data) != nil {
		log.Log(log.Error, "[NATS] failed to publish vote for %s", prop.ID)
	}
}

func handleVote(m *nats.Msg) {
	var v Vote
	if err := json.Unmarshal(m.Data, &v); err != nil {
		log.Log(log.Error, "[NATS] handleVote: unmarshal error: %v", err)
		return
	}
	markNodeHeard(v.SenderNodeID)

	State.Mu.Lock()
	pt, ok := State.Proposals[v.ProposalID]
	if !ok || pt.Finalized {
		State.Mu.Unlock()
		return
	}
	pt.Votes[v.NodeID] = v.Agree
	decideLocked(pt)
	State.Mu.Unlock()
}

func decideLocked(pt *ProposalTracking) {
	total := countActiveMonitorsLocked()
	if total == 0 {
		return
	}
	maj := (total / 2) + 1

	yes, no := 0, 0
	for nid, agree := range pt.Votes {
		if node, ok := State.ClusterNodes[nid]; ok && node.NodeRole == "IBPMonitor" && isNodeActive(node) {
			if agree {
				yes++
			} else {
				no++
			}
		}
	}

	switch {
	case yes >= maj && yes >= minConsensusVotes:
		pt.Finalized, pt.Passed = true, true
	case no >= maj && no >= minConsensusVotes:
		pt.Finalized, pt.Passed = true, false
	}

	if pt.Finalized {
		if pt.Timer != nil {
			pt.Timer.Stop()
		}
		go finalize(pt)
	}
}

/*─────────────────────────────────────────────────────────────
  FINALISATION (timer fallback + explicit)
─────────────────────────────────────────────────────────────*/

func forceFinalize(pid ProposalID) {
	State.Mu.Lock()
	pt, ok := State.Proposals[pid]
	if !ok || pt.Finalized {
		State.Mu.Unlock()
		return
	}
	decideLocked(pt)
	State.Mu.Unlock()
}

func finalize(pt *ProposalTracking) {
	msg := FinalizeMessage{
		Proposal:  pt.Proposal,
		Passed:    pt.Passed,
		DecidedAt: time.Now().UTC(),
	}
	if data, _ := json.Marshal(msg); Publish(State.SubjectFinalize, data) != nil {
		log.Log(log.Error, "[NATS] failed to publish finalize for %s", pt.Proposal.ID)
	}

	if pt.Passed {
		applyOfficialChanges(pt.Proposal)
	}

	State.Mu.Lock()
	delete(State.Proposals, pt.Proposal.ID)
	State.Mu.Unlock()
}

func handleFinalize(m *nats.Msg) {
	var fm FinalizeMessage
	if err := json.Unmarshal(m.Data, &fm); err != nil {
		log.Log(log.Error, "[NATS] handleFinalize: unmarshal error: %v", err)
		return
	}
	markNodeHeard(fm.Proposal.SenderNodeID)

	if fm.Passed {
		applyOfficialChanges(fm.Proposal)
	}
}

/*─────────────────────────────────────────────────────────────
  APPLY TO OFFICIAL SNAPSHOT
─────────────────────────────────────────────────────────────*/

func applyOfficialChanges(prop Proposal) {
	chk, okChk := findCheckByName(prop.CheckName, prop.CheckType)
	if !okChk {
		log.Log(log.Warn, "[NATS] applyOfficialChanges: check %s/%s not found", prop.CheckType, prop.CheckName)
		return
	}
	mem, okMem := findMemberByName(prop.MemberName)
	if !okMem {
		log.Log(log.Warn, "[NATS] applyOfficialChanges: member %s not found", prop.MemberName)
		return
	}

	var svc cfg.Service
	if prop.CheckType == "domain" || prop.CheckType == "endpoint" {
		s, ok := findServiceForDomain(prop.DomainName)
		if ok {
			svc = s
		}
	}

	switch prop.CheckType {
	case "site":
		dat.UpdateOfficialSiteResult(chk, mem, prop.ProposedStatus, prop.ErrorText, prop.Data, prop.IsIPv6)
	case "domain":
		dat.UpdateOfficialDomainResult(chk, mem, svc, prop.DomainName, prop.ProposedStatus, prop.ErrorText, prop.Data, prop.IsIPv6)
	case "endpoint":
		dat.UpdateOfficialEndpointResult(chk, mem, svc, prop.DomainName, prop.Endpoint, prop.ProposedStatus, prop.ErrorText, prop.Data, prop.IsIPv6)
	}
}

/*─────────────────────────────────────────────────────────────
  HELPER – LOCAL STATUS LOOK‑UP
─────────────────────────────────────────────────────────────*/

func checkLocalStatus(checkType, checkName, memberName, domainName, endpoint string, isIPv6 bool) (bool, bool) {
	switch checkType {
	case "site":
		return dat.GetLocalSiteStatusIPv4v6(checkName, memberName, isIPv6)
	case "domain":
		return dat.GetLocalDomainStatusIPv4v6(checkName, memberName, domainName, isIPv6)
	case "endpoint":
		return dat.GetLocalEndpointStatusIPv4v6(checkName, memberName, domainName, endpoint, isIPv6)
	default:
		return false, false
	}
}

/*─────────────────────────────────────────────────────────────
  HELPER – LIVE MONITOR COUNT (LOCK HELD BY CALLER)
─────────────────────────────────────────────────────────────*/

func countActiveMonitorsLocked() int {
	n := 0
	for _, node := range State.ClusterNodes {
		if node.NodeRole == "IBPMonitor" && isNodeActive(node) {
			n++
		}
	}
	return n
}
