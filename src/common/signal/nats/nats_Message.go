package nats

import "github.com/nats-io/nats.go"

func Message(subject string, data []byte) error {
	natsMu.Lock()
	defer natsMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nats.ErrConnectionClosed
	}
	return nc.Publish(subject, data)
}
