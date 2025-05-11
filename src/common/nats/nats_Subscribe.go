package nats

import (
	"github.com/nats-io/nats.go"
)

func Subscribe() error {
	var sub *nats.Subscription
	var err error

	sub, err = nc.Subscribe(state.SubjectPropose, handleProposal)
	if err != nil {
		return err
	}
	sub.SetPendingLimits(-1, 268435456)

	sub, err = nc.Subscribe(state.SubjectVote, handleVote)
	if err != nil {
		return err
	}
	sub.SetPendingLimits(-1, 268435456)

	sub, err = nc.Subscribe(state.SubjectFinalize, handleFinalize)
	if err != nil {
		return err
	}
	sub.SetPendingLimits(-1, 268435456)

	return nil
}
