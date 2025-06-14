package nats

import (
	"encoding/json"
	"time"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	data2 "ibp-geodns/src/common/data2"
	log "ibp-geodns/src/common/logging"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

const minConsensusVotes = 2

func ProposeCheckStatus(
	checkType, checkName, memberName,
	domainName, endpoint string,
	status bool,
	errorText string,
	dataMap map[string]interface{},
	isIPv6 bool,
) {
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

	log.Log(log.Debug,
		"[CONSENSUS] → PROPOSAL created id=%s type=%s member=%s status=%v v6=%v",
		prop.ID, prop.CheckType, prop.MemberName, prop.ProposedStatus, prop.IsIPv6)
	log.Log(log.Debug, "[CONSENSUS]     details=%+v", prop)

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

func handleProposal(m *nats.Msg) {
	var prop Proposal
	if err := json.Unmarshal(m.Data, &prop); err != nil {
		log.Log(log.Error, "[NATS] handleProposal: unmarshal error: %v", err)
		return
	}
	log.Log(log.Debug,
		"[CONSENSUS] ← PROPOSAL received id=%s type=%s member=%s status=%v v6=%v",
		prop.ID, prop.CheckType, prop.MemberName, prop.ProposedStatus, prop.IsIPv6)
	markNodeHeard(prop.SenderNodeID)

	State.Mu.Lock()
	if _, exists := State.Proposals[prop.ID]; !exists {
		State.Proposals[prop.ID] = &ProposalTracking{
			Proposal: prop,
			Votes:    make(map[string]bool),
		}
		State.Proposals[prop.ID].Timer = time.AfterFunc(State.ProposalTimeout,
			func() { forceFinalize(prop.ID) })
		State.Mu.Unlock()
		go voteOnProposal(prop)
		return
	}
	State.Mu.Unlock()
}

func voteOnProposal(prop Proposal) {
	time.Sleep(5 * time.Millisecond)

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

	log.Log(log.Debug,
		"[CONSENSUS]    vote id=%s agree=%v (local=%v proposed=%v)",
		prop.ID, v.Agree, localStatus, prop.ProposedStatus)

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
	log.Log(log.Debug, "[CONSENSUS] ← vote id=%s from=%s agree=%v", v.ProposalID, v.NodeID, v.Agree)
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
		log.Log(log.Info,
			"[CONSENSUS] ⇢ finalize id=%s PASS=%v yes=%d no=%d (%d active monitors)",
			pt.Proposal.ID, pt.Passed, yes, no, total)

		if pt.Timer != nil {
			pt.Timer.Stop()
		}
		go finalize(pt)
	}
}

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
	log.Log(log.Debug,
		"[CONSENSUS] ← FINALIZE id=%s PASS=%v", fm.Proposal.ID, fm.Passed)
	markNodeHeard(fm.Proposal.SenderNodeID)

	if fm.Passed && State.ThisNode.NodeRole == "IBPMonitor" {
		applyOfficialChanges(fm.Proposal)

	} else if fm.Passed && State.ThisNode.NodeRole == "IBPCollator" {
		ct := checkTypeToInt(fm.Proposal.CheckType)
		url := deriveCheckURL(fm.Proposal)

		rec := data2.NetStatusRecord{
			CheckType: ct,
			CheckName: fm.Proposal.CheckName,
			CheckURL:  url,
			Domain:    fm.Proposal.DomainName,
			Member:    fm.Proposal.MemberName,
			IsIPv6:    fm.Proposal.IsIPv6,
		}

		if !fm.Proposal.ProposedStatus {
			rec.Status = false
			rec.StartTime = fm.DecidedAt.UTC()
			rec.Error = fm.Proposal.ErrorText
			rec.VoteData = nil
			rec.Extra = fm.Proposal.Data

			if err := data2.InsertNetStatus(rec); err != nil {
				log.Log(log.Error, "[NATS] handleFinalize: InsertNetStatus: %v", err)
			}
		} else {
			if err := data2.CloseOpenEvent(rec); err != nil {
				log.Log(log.Error, "[NATS] handleFinalize: CloseOpenEvent: %v", err)
			}
		}
	}
}

func applyOfficialChanges(prop Proposal) {
	log.Log(log.Debug,
		"[CONSENSUS] ⇢ apply official change id=%s type=%s member=%s status=%v v6=%v",
		prop.ID, prop.CheckType, prop.MemberName, prop.ProposedStatus, prop.IsIPv6)

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

func countActiveMonitorsLocked() int {
	n := 0
	for _, node := range State.ClusterNodes {
		if node.NodeRole == "IBPMonitor" && isNodeActive(node) {
			n++
		}
	}
	return n
}

func checkTypeToInt(t string) int {
	switch t {
	case "site":
		return 1
	case "domain":
		return 2
	case "endpoint":
		return 3
	default:
		return 0
	}
}

func deriveCheckURL(p Proposal) string {
	switch p.CheckType {
	case "endpoint":
		return p.Endpoint
	case "domain":
		return p.DomainName
	default:
		return ""
	}
}
