package nats

import (
	"encoding/json"
	"time"

	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

// ProposeCheckStatus handles creating a new proposal for site/domain/endpoint status.
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
			log.Log(log.Debug, "Propose skipped: Active proposal already exists for CheckType=%s, CheckName=%s, MemberName=%s", checkType, checkName, memberName)
			return false
		}
	}
	State.Mu.RUnlock()

	Propose(checkType, checkName, memberName, domainName, endpoint, status, errorText, dataMap)
	return true
}

// Propose generates a new Proposal, stores it in State, and publishes it.
func Propose(checkType string, checkName string, memberName string, domainName string, endpoint string, proposedStatus bool, errorText string, dataMap map[string]interface{}) (ProposalID, error) {
	pid := ProposalID(generateProposalID())
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

// finalizeVote finalizes a proposal due to a timeout or majority.
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

	var finalStatus bool
	if yesCount >= majority {
		finalStatus = true
	} else if noCount >= majority {
		finalStatus = false
	} else {
		finalStatus = (yesCount > noCount)
	}

	pt.FinalStatus = finalStatus
	State.Mu.Unlock()

	log.Log(log.Debug, "Proposal timeout or forced finalize: ProposalID=%s, FinalStatus=%t", pid, finalStatus)

	if finalStatus {
		go applyOfficialChanges(pt.Proposal)
		go PublishFinalize(FinalizeMessage{
			ProposalID:  pid,
			FinalStatus: true,
			DecidedAt:   time.Now().UTC(),
		})
	} else {
		go PublishFinalize(FinalizeMessage{
			ProposalID:  pid,
			FinalStatus: false,
			DecidedAt:   time.Now().UTC(),
		})
	}
}

// handleProposal receives a new proposal from another monitor.
func handleProposal(m *nats.Msg) {
	var prop Proposal
	if err := json.Unmarshal(m.Data, &prop); err != nil {
		log.Log(log.Error, "Failed to unmarshal proposal message: %v", err)
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

	go func(prop Proposal) {
		found, localStatus := checkLocalStatus(prop.CheckType, prop.CheckName, prop.MemberName, prop.DomainName, prop.Endpoint)
		if !found {
			return
		}
		v := Vote{
			ProposalID: prop.ID,
			NodeID:     State.NodeID,
			Agree:      (localStatus == prop.ProposedStatus),
			Timestamp:  time.Now().UTC(),
		}
		data, _ := json.Marshal(v)
		go Publish(State.SubjectVote, data)
	}(prop)
}

// handleVote receives a vote from another monitor.
func handleVote(m *nats.Msg) {
	var vote Vote
	if err := json.Unmarshal(m.Data, &vote); err != nil {
		log.Log(log.Error, "Failed to unmarshal vote message: %v", err)
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

// handleFinalize applies the final decision from another monitor.
func handleFinalize(m *nats.Msg) {
	var fm FinalizeMessage
	if err := json.Unmarshal(m.Data, &fm); err != nil {
		log.Log(log.Error, "Failed to unmarshal finalize message: %v", err)
		return
	}

	State.Mu.RLock()
	pt, exists := State.Proposals[fm.ProposalID]
	State.Mu.RUnlock()
	if exists && !pt.Finalized {
		State.Mu.Lock()
		State.Proposals[pt.Proposal.ID].Finalized = true
		State.Mu.Unlock()
		go finalizeVote(pt.Proposal.ID)
	}
}

// applyOfficialChanges updates official results once a vote passes.
func applyOfficialChanges(proposal Proposal) {
	// Insert the code that modifies official site/domain/endpoint status
	// in data/Official or your final store. This can call data.UpdateOfficialSiteResult,
	// data.UpdateOfficialDomainResult, etc.
	// Omitted code is presumably found in the existing code base.
	log.Log(log.Info, "Finalizing official status for proposal: %s, status=%t", proposal.ID, proposal.ProposedStatus)
	// ...existing logic...
}
