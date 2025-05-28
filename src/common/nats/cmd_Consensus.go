package nats

import (
	"encoding/json"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	// For local checks and official updates
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
			log.Log(log.Debug, "Propose skipped: identical active proposal already exists for CheckType=%s, CheckName=%s, Member=%s", checkType, checkName, memberName)
			return false
		}
	}
	State.Mu.RUnlock()

	Propose(checkType, checkName, memberName, domainName, endpoint, status, errorText, dataMap)
	return true
}

// Propose creates a new proposal, stores it, and publishes
func Propose(checkType, checkName, memberName, domainName, endpoint string, proposedStatus bool, errorText string, dataMap map[string]interface{}) (ProposalID, error) {
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

	// evaluate local status
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
	totalNodes := len(State.ClusterNodes)
	majority := (totalNodes / 2) + 1
	if majority < 2 {
		majority = 2
	}

	yesCount := 0
	noCount := 0
	for _, v := range pt.Votes {
		if v {
			yesCount++
		} else {
			noCount++
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

	if len(pt.Votes) == totalNodes && !pt.Finalized {
		pt.Finalized = true
		State.Mu.Unlock()
		finalizeVote(vote.ProposalID)
		return
	}
	State.Mu.Unlock()
}

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

	totalNodes := len(State.ClusterNodes)
	majority := (totalNodes / 2) + 1

	yesCount := 0
	noCount := 0
	for _, v := range pt.Votes {
		if v {
			yesCount++
		} else {
			noCount++
		}
	}
	finalStatus := false
	if yesCount >= majority {
		finalStatus = true
	} else if noCount >= majority {
		finalStatus = false
	} else {
		finalStatus = (yesCount > noCount)
	}
	pt.FinalStatus = finalStatus
	State.Mu.Unlock()

	if finalStatus {
		// apply official up
		go applyOfficialChanges(pt.Proposal, true)
		_ = PublishFinalize(FinalizeMessage{
			ProposalID:  pid,
			FinalStatus: true,
			DecidedAt:   time.Now().UTC(),
		})
	} else {
		// apply official down
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
	// Use data from data/results_Local.go
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
	// Retrieve the needed config data: check, member, and possibly service
	chk, chkExists := findCheckByName(prop.CheckName, prop.CheckType)
	if !chkExists {
		log.Log(log.Warn, "applyOfficialChanges: no check named %s (type=%s) found", prop.CheckName, prop.CheckType)
		return
	}
	mem, memExists := findMemberByName(prop.MemberName)
	if !memExists {
		log.Log(log.Warn, "applyOfficialChanges: no member named %s found", prop.MemberName)
		return
	}

	var svc cfg.Service
	if prop.CheckType == "domain" || prop.CheckType == "endpoint" {
		s, sExists := findServiceForDomain(prop.DomainName)
		if !sExists && prop.CheckType == "domain" {
			log.Log(log.Warn, "applyOfficialChanges: domain service not found for %s", prop.DomainName)
			return
		}
		svc = s
	}

	status := final
	errMsg := prop.ErrorText
	dataMap := prop.Data

	switch prop.CheckType {
	case "site":
		log.Log(log.Info, "Finalizing site check for member=%s => %t", prop.MemberName, status)
		dat.UpdateOfficialSiteResult(chk, mem, status, errMsg, dataMap)

	case "domain":
		log.Log(log.Info, "Finalizing domain check for member=%s => %t domain=%s", prop.MemberName, status, prop.DomainName)
		dat.UpdateOfficialDomainResult(chk, mem, svc, prop.DomainName, status, errMsg, dataMap)

	case "endpoint":
		log.Log(log.Info, "Finalizing endpoint check for member=%s => %t domain=%s endpoint=%s", prop.MemberName, status, prop.DomainName, prop.Endpoint)
		dat.UpdateOfficialEndpointResult(chk, mem, svc, prop.DomainName, prop.Endpoint, status, errMsg, dataMap)

	default:
		log.Log(log.Warn, "applyOfficialChanges: unknown checkType %s", prop.CheckType)
	}
}
