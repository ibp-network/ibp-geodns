package nats

import (
	log "ibp-geodns/src/common/logging"

	"ibp-geodns/src/common/signal/types"
)

// finalizeDueToTimeout: Handles proposals that timed out.
func finalizeVote(pid types.ProposalID, state types.NodeState) {
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
	log.Log(log.Debug, "Proposal timeout: ProposalID=%s, FinalStatus=%t", pid, finalStatus)
	if finalStatus {
		// Do stuff if a proposal for official change is passed
		go applyOfficialChanges(pt.Proposal)
		go Finalize(pid, finalStatus, state)
	}
}
