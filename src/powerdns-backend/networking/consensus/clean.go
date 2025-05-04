package consensus

import (
	"time"
)

// Timer to run clean up task for old proposals every 5 seconds.
func StartProposalCleanup() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			CleanOldProposals()
		}
	}()
}

// CleanOldProposals removes proposals from state.Proposals that are older than 10 seconds.
func CleanOldProposals() {
	state.Mu.Lock()
	defer state.Mu.Unlock()

	now := time.Now().UTC()
	threshold := 6 * time.Second

	for pid, pt := range state.Proposals {
		// Check the age of the proposal
		if now.Sub(pt.Proposal.Timestamp) > threshold {
			// Log and remove old proposal
			delete(state.Proposals, pid)

			// Stop the timer for this proposal, if active
			if pt.Timer != nil {
				pt.Timer.Stop()
			}
		}
	}
}
