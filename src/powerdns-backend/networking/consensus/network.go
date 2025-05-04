package consensus

import (
	"encoding/json"
	"ibp-geodns/config"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

var (
	nc     *nats.Conn
	natsMu sync.Mutex
)

func ConnectNats(url string) error {
	c := config.GetConfig()
	natsMu.Lock()
	defer natsMu.Unlock()

	if nc != nil && !nc.IsClosed() {
		return nil
	}

	var err error
	nc, err = nats.Connect(url, nats.UserInfo(c.System.Nats.User, c.System.Nats.Pass))
	return err
}

func CloseNats() {
	natsMu.Lock()
	defer natsMu.Unlock()
	if nc != nil && !nc.IsClosed() {
		nc.Close()
		nc = nil
	}
}

func subscribeSubjects() error {
	var sub *nats.Subscription
	var err error

	sub, err = nc.Subscribe(state.SubjectPropose, handleProposeMessage)
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

func publishMessage(subject string, data []byte) error {
	natsMu.Lock()
	defer natsMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nats.ErrConnectionClosed
	}
	return nc.Publish(subject, data)
}

func publishFinalize(pid ProposalID, finalStatus bool) {
	fm := FinalizeMessage{
		ProposalID:  pid,
		FinalStatus: finalStatus,
		DecidedAt:   time.Now().UTC(),
	}
	data, _ := json.Marshal(fm)
	publishMessage(state.SubjectFinalize, data)
}
