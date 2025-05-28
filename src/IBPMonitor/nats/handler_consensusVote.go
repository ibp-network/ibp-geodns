package nats

import (
	"encoding/json"

	"github.com/nats-io/nats.go"

	log "ibp-geodns/src/common/logging"
)

// handleProposeMessage: Receives a proposal message and processes it.
func handleVote(m *nats.Msg) {
	var vote Vote
	if err := json.Unmarshal(m.Data, &vote); err != nil {
		log.Log(log.Error, "Failed to unmarshal vote message: %v", err)
		return
	}

	// Log that a vote was received
	//l.Log(l.Debug, "Received vote: ProposalID=%s, NodeID=%s, Agree=%t", vote.ProposalID, vote.NodeID, vote.Agree)

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
