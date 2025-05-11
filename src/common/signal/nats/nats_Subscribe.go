package nats

import (
	"github.com/nats-io/nats.go"
)

func Subscribe(state NodeState) error {
	var sub *nats.Subscription
	var err error

	sub, err = nc.Subscribe(state.SubjectPropose, handleProposalMessage)
	if err != nil {
		return err
	}
	sub.SetPendingLimits(-1, 268435456)

	sub, err = nc.Subscribe(state.SubjectVote, handleVoteMessage)
	if err != nil {
		return err
	}
	sub.SetPendingLimits(-1, 268435456)

	sub, err = nc.Subscribe(state.SubjectFinalize, handleFinalizeMessage)
	if err != nil {
		return err
	}
	sub.SetPendingLimits(-1, 268435456)

	return nil
}
