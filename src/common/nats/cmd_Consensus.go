package nats

import (
	"encoding/json"
	"time"

	log "ibp-geodns/src/common/logging"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
)

// ProposeCheckStatus is a helper to propose a site/domain/endpoint status
func ProposeCheckStatus(checkType, checkName, memberName, domainName, endpoint string, status bool, errorText string, dataMap map[string]interface{}) bool {
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
			log.Log(log.Debug, "Propose skipped: identical active proposal already exists for CheckType=%s, CheckName=%s, Member=%s",
				checkType, checkName, memberName)
			return false
		}
	}
	State.Mu.RUnlock()

	Propose(checkType, checkName, memberName, domainName, endpoint, status, errorText, dataMap)
	return true
}

// Propose creates a new proposal, stores it, and publishes it to 'consensus.propose'
func Propose(
	checkType, checkName, memberName,
	domainName, endpoint string,
	proposedStatus bool,
	errorText string,
	dataMap map[string]interface{},
) (ProposalID, error) {

	// Use google/uuid for the proposal ID
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
		Data:           dataMap,
		Timestamp:      time.Now().UTC(),
	}

	pt := &ProposalTracking{
		Proposal: prop,
		Votes:    make(map[string]bool),
	}

	State.Mu.Lock()
	State.Proposals[pid] = pt
	State.Mu.Unlock()

	pt.Timer = time.AfterFunc(State.ProposalTimeout, func() {
		finalizeVote(pid)
	})

	data, _ := json.Marshal(prop)
	err := Publish(State.SubjectPropose, data)
	return pid, err
}

// handleProposal processes incoming "consensus.propose" messages.
func handleProposal(m *nats.Msg) {
	var prop Proposal
	if err := json.Unmarshal(m.Data, &prop); err != nil {
		log.Log(log.Error, "handleProposal: unmarshal error: %v", err)
		return
	}

	State.Mu.Lock()
	_, exists := State.Proposals[prop.ID]
	if !exists {
		pt := &ProposalTracking{
			Proposal: prop,
			Votes:    make(map[string]bool),
		}
		State.Proposals[prop.ID] = pt
		pt.Timer = time.AfterFunc(State.ProposalTimeout, func() {
			finalizeVote(prop.ID)
		})
	}
	State.Mu.Unlock()

	go func(pr Proposal) {
		found, localStatus := checkLocalStatus(pr.CheckType, pr.CheckName, pr.MemberName, pr.DomainName, pr.Endpoint)
		if !found {
			return
		}
		v := Vote{
			ProposalID: pr.ID,
			NodeID:     State.NodeID,
			Agree:      (localStatus == pr.ProposedStatus),
			Timestamp:  time.Now().UTC(),
		}
		data, _ := json.Marshal(v)
		_ = Publish(State.SubjectVote, data)
	}(prop)
}

// handleVote processes incoming "consensus.vote" messages.
func handleVote(m *nats.Msg) {
	var vote Vote
	if err := json.Unmarshal(m.Data, &vote); err != nil {
		log.Log(log.Error, "handleVote: unmarshal error: %v", err)
		return
	}

	State.Mu.Lock()
	pt, exists := State.Proposals[vote.ProposalID]
	if !exists || pt.Finalized {
		State.Mu.Unlock()
		return
	}

	pt.Votes[vote.NodeID] = vote.Agree

	monitorCount := CountMonitorNodes()
	majority := (monitorCount / 2) + 1

	yesCount := 0
	noCount := 0
	for nodeID, v := range pt.Votes {
		node, inMap := State.ClusterNodes[nodeID]
		if inMap && node.NodeRole == "monitor" {
			if v {
				yesCount++
			} else {
				noCount++
			}
		}
	}

	if yesCount >= majority && !pt.Finalized {
		pt.Finalized = true
		State.Mu.Unlock()
		finalizeVote(vote.ProposalID)
		return
	} else if noCount >= majority && !pt.Finalized {
		pt.Finalized = true
		State.Mu.Unlock()
		finalizeVote(vote.ProposalID)
		return
	}

	if len(pt.Votes) == monitorCount && !pt.Finalized {
		pt.Finalized = true
		State.Mu.Unlock()
		finalizeVote(vote.ProposalID)
		return
	}

	State.Mu.Unlock()
}

// handleFinalize processes incoming "consensus.finalize" messages.
func handleFinalize(m *nats.Msg) {
	var fm FinalizeMessage
	if err := json.Unmarshal(m.Data, &fm); err != nil {
		log.Log(log.Error, "handleFinalize: unmarshal error: %v", err)
		return
	}

	State.Mu.RLock()
	pt, exists := State.Proposals[fm.ProposalID]
	State.Mu.RUnlock()
	if exists && !pt.Finalized {
		State.Mu.Lock()
		pt.Finalized = true
		State.Mu.Unlock()
		go finalizeVote(pt.Proposal.ID)
	}
}

// finalizeVote is called when a proposal times out or is forced to finalize.
func finalizeVote(pid ProposalID) {
	State.Mu.Lock()
	pt, exists := State.Proposals[pid]
	if !exists {
		State.Mu.Unlock()
		return
	}
	if pt.Timer != nil {
		pt.Timer.Stop()
	}
	monitorCount := CountMonitorNodes()

	yesCount := 0
	noCount := 0
	for nodeID, v := range pt.Votes {
		node, inMap := State.ClusterNodes[nodeID]
		if inMap && node.NodeRole == "monitor" {
			if v {
				yesCount++
			} else {
				noCount++
			}
		}
	}
	majority := (monitorCount / 2) + 1

	var finalStatus bool
	if yesCount >= majority {
		finalStatus = true
	} else if noCount >= majority {
		finalStatus = false
	} else {
		finalStatus = (yesCount > noCount)
	}
	pt.FinalStatus = finalStatus
	pt.Finalized = true
	State.Mu.Unlock()

	if finalStatus {
		go applyOfficialChanges(pt.Proposal, true)
		_ = PublishFinalize(FinalizeMessage{
			ProposalID:  pid,
			FinalStatus: true,
			DecidedAt:   time.Now().UTC(),
		})
	} else {
		go applyOfficialChanges(pt.Proposal, false)
		_ = PublishFinalize(FinalizeMessage{
			ProposalID:  pid,
			FinalStatus: false,
			DecidedAt:   time.Now().UTC(),
		})
	}

	log.Log(log.Info, "Proposal %s finalized => status=%v", pid, finalStatus)
}

// checkLocalStatus consults local data about site/domain/endpoint
func checkLocalStatus(checkType, checkName, memberName, domainName, endpoint string) (bool, bool) {
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
		log.Log(log.Warn, "applyOfficialChanges: no check named %s (type=%s)", prop.CheckName, prop.CheckType)
		return
	}
	mem, memOk := findMemberByName(prop.MemberName)
	if !memOk {
		log.Log(log.Warn, "applyOfficialChanges: no member named %s found", prop.MemberName)
		return
	}
	var svc cfg.Service
	if prop.CheckType == "domain" || prop.CheckType == "endpoint" {
		serviceObj, ok := findServiceForDomain(prop.DomainName)
		if !ok && prop.CheckType == "domain" {
			log.Log(log.Warn, "applyOfficialChanges: domain service not found for domain=%s", prop.DomainName)
			return
		}
		svc = serviceObj
	}
	status := final
	errorMsg := prop.ErrorText
	dataMap := prop.Data

	switch prop.CheckType {
	case "site":
		log.Log(log.Info, "Finalizing site check for member=%s => %t", prop.MemberName, status)
		dat.UpdateOfficialSiteResult(chk, mem, status, errorMsg, dataMap)
	case "domain":
		log.Log(log.Info, "Finalizing domain check for member=%s => %t domain=%s", prop.MemberName, status, prop.DomainName)
		dat.UpdateOfficialDomainResult(chk, mem, svc, prop.DomainName, status, errorMsg, dataMap)
	case "endpoint":
		log.Log(log.Info, "Finalizing endpoint check for member=%s => %t domain=%s endpoint=%s",
			prop.MemberName, status, prop.DomainName, prop.Endpoint)
		dat.UpdateOfficialEndpointResult(chk, mem, svc, prop.DomainName, prop.Endpoint, status, errorMsg, dataMap)
	default:
		log.Log(log.Warn, "applyOfficialChanges: unrecognized checkType=%s", prop.CheckType)
	}
}
