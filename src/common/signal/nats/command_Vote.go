package nats

import (
	"encoding/json"
	"ibp-geodns/src/common/signal/types"

	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

// handleVoteMessage: Processes votes and may finalize early.
func handleVoteMessage(m *nats.Msg, state types.NodeState) {
	var vote types.Vote
	if err := json.Unmarshal(m.Data, &vote); err != nil {
		log.Log(log.Error, "Failed to unmarshal vote message: %v", err)
		return
	}

	// Log that a vote was received
	log.Log(log.Debug, "Received vote: ProposalID=%s, NodeID=%s, Agree=%t", vote.ProposalID, vote.NodeID, vote.Agree)

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
