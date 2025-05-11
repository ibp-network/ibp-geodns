package nats

import (
	"encoding/json"
	log "ibp-geodns/src/common/logging"

	"ibp-geodns/src/common/signal/types"

	"github.com/nats-io/nats.go"
)

// handleFinalizeMessage: Applies the final decision.
func handleFinalizeMessage(m *nats.Msg, state types.NodeState) {
	var fm types.FinalizeMessage
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
		go finalizeVote(pt.Proposal.ID, state)
	}
}
