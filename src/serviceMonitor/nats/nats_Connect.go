package nats

import (
	cfg "ibp-geodns/src/common/config"

	nats "github.com/nats-io/nats.go"
)

func Connect() error {
	c := cfg.GetConfig()
	natsMu.Lock()
	defer natsMu.Unlock()

	if nc != nil && !nc.IsClosed() {
		return nil
	}

	var err error
	nc, err = nats.Connect(c.Local.Signal.Url, nats.UserInfo(c.Local.Signal.User, c.Local.Signal.Pass))
	return err
}
