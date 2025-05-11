package nats

import (
	"encoding/json"
	log "ibp-geodns/src/common/logging"
	"time"

	"github.com/nats-io/nats.go"
)

// handleProposeMessage: Receives a proposal message and processes it.
func handleProposalMessage(m *nats.Msg, state NodeState) {
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
			finalizeVote(prop.ID, state)
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
		log.Log(log.Debug, "Voting on proposal: ID=%s, Agree=%t, NodeID=%s", prop.ID, v.Agree, state.NodeID)

		data, _ := json.Marshal(v)
		go Message(state.SubjectVote, data)
	}(prop)
}
