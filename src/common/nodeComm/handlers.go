package nodeComm

import (
	"encoding/json"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
	"time"

	"github.com/nats-io/nats.go"
)

// handleProposeMessage: Receives a proposal message and processes it.
func handleProposeMessage(m *nats.Msg) {
	var prop Proposal
	if err := json.Unmarshal(m.Data, &prop); err != nil {
		log.Log(log.Error, "Failed to unmarshal proposal message: %v", err)
		return
	}

	// Log that the proposal was received
	//log.Log(log.Debug, "Received proposal: ID=%s, CheckType=%s, MemberName=%s", prop.ID, prop.CheckType, prop.MemberName)

	state.Mu.Lock()
	_, exists := state.Proposals[prop.ID]
	if !exists {
		pt := &ProposalTracking{
			Proposal: prop,
			Votes:    make(map[string]bool),
		}

		state.Proposals[prop.ID] = pt
		pt.Timer = time.AfterFunc(state.ProposalTimeout, func() {
			finalizeVote(prop.ID)
		})
	}
	state.Mu.Unlock()

	go func(prop Proposal) {
		found, localStatus := checkLocalStatus(prop.CheckType, prop.CheckName, prop.MemberName, prop.DomainName, prop.Endpoint)
		if !found {
			// Not participating in voting because we don't have a local state for this check.
			return
		}

		v := Vote{
			ProposalID: prop.ID,
			NodeID:     state.NodeID,
			Agree:      (localStatus == prop.ProposedStatus),
			Timestamp:  time.Now().UTC(),
		}

		// Log that the node is voting
		//log.Log(log.Debug, "Voting on proposal: ID=%s, Agree=%t, NodeID=%s", prop.ID, v.Agree, state.NodeID)

		data, _ := json.Marshal(v)
		go publishMessage(state.SubjectVote, data)
	}(prop)
}

// handleVoteMessage: Processes votes and may finalize early.
func handleVoteMessage(m *nats.Msg) {
	var vote Vote
	if err := json.Unmarshal(m.Data, &vote); err != nil {
		log.Log(log.Error, "Failed to unmarshal vote message: %v", err)
		return
	}

	// Log that a vote was received
	//log.Log(log.Debug, "Received vote: ProposalID=%s, NodeID=%s, Agree=%t", vote.ProposalID, vote.NodeID, vote.Agree)

	state.Mu.Lock()
	pt, exists := state.Proposals[vote.ProposalID]
	if !exists || pt.Finalized {
		state.Mu.Unlock()
		return
	}

	pt.Votes[vote.NodeID] = vote.Agree

	totalNodes := len(state.ClusterNodes)
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

	// Early finalize if majority reached
	if yesCount >= majority && !pt.Finalized {
		state.Proposals[pt.Proposal.ID].Finalized = true
		state.Mu.Unlock()
		go finalizeVote(vote.ProposalID)
		return
	} else if noCount >= majority && !pt.Finalized {
		state.Proposals[pt.Proposal.ID].Finalized = true
		state.Mu.Unlock()
		go finalizeVote(vote.ProposalID)
		return
	}

	// If all nodes voted, finalize
	if len(pt.Votes) == totalNodes && !pt.Finalized {
		state.Proposals[pt.Proposal.ID].Finalized = true
		state.Mu.Unlock()
		go finalizeVote(vote.ProposalID)
		return
	}

	state.Mu.Unlock()
}

// handleFinalizeMessage: Applies the final decision.
func handleFinalizeMessage(m *nats.Msg) {
	var fm FinalizeMessage
	if err := json.Unmarshal(m.Data, &fm); err != nil {
		log.Log(log.Error, "Failed to unmarshal finalize message: %v", err)
		return
	}

	state.Mu.RLock()
	pt, exists := state.Proposals[fm.ProposalID]
	state.Mu.RUnlock()

	if exists && !pt.Finalized {
		state.Mu.Lock()
		state.Proposals[pt.Proposal.ID].Finalized = true
		state.Mu.Unlock()
		go finalizeVote(pt.Proposal.ID)
	}
}

// finalizeDueToTimeout: Handles proposals that timed out.
func finalizeVote(pid ProposalID) {
	state.Mu.Lock()
	pt, exists := state.Proposals[pid]
	if !exists {
		state.Mu.Unlock()
		return
	}

	if pt.Timer != nil {
		pt.Timer.Stop()
	}

	totalNodes := len(state.ClusterNodes)
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

	// Decide based on what we have
	finalStatus := false
	if yesCount >= majority {
		finalStatus = true
	} else if noCount >= majority {
		finalStatus = false
	} else {
		finalStatus = (yesCount > noCount)
	}

	pt.FinalStatus = finalStatus

	state.Mu.Unlock()

	// Log the timeout and final status
	//log.Log(log.Debug, "Proposal timeout: ProposalID=%s, FinalStatus=%t", pid, finalStatus)
	if finalStatus {
		// Do stuff if a proposal for official change is passed
		go applyOfficialChanges(pt.Proposal)
		go publishFinalize(pid, finalStatus)
	}
}

// checkLocalStatus returns this node's local perspective: true=online, false=offline.
func checkLocalStatus(checkType string, checkName string, memberName string, domainName string, endpoint string) (bool, bool) {
	switch checkType {
	case "site":
		found, status := dat.GetLocalSiteStatus(checkName, memberName)
		if !found {
			return false, false
		}
		return true, status

	case "domain":
		found, status := dat.GetLocalDomainStatus(checkName, memberName, domainName)
		if !found {
			return false, false
		}
		return true, status

	case "endpoint":
		found, status := dat.GetLocalEndpointStatus(checkName, memberName, endpoint)
		if !found {
			return false, false
		}
		return true, status

	default:
		log.Log(log.Warn, "checkLocalStatus: unknown checkType %s, defaulting to offline", checkType)
		return false, false
	}
}
