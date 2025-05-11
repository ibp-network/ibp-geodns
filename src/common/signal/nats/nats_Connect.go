package nats

import (
	cfg "ibp-geodns/src/common/config"

	"github.com/nats-io/nats.go"
)

func Connect(url string) error {
	c := cfg.GetConfig()
	natsMu.Lock()
	defer natsMu.Unlock()

	if nc != nil && !nc.IsClosed() {
		return nil
	}

	var err error
	nc, err = nats.Connect(url, nats.UserInfo(c.Local.Signal.User, c.Local.Signal.Pass))
	return err
}
