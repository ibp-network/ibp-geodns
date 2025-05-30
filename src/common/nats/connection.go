package nats

import (
	"fmt"
	"sync"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

var (
	nc           *nats.Conn
	connectionMu sync.Mutex
)

func Connect() error {
	connectionMu.Lock()
	defer connectionMu.Unlock()

	if nc != nil && !nc.IsClosed() {
		log.Log(log.Debug, "[NATS] Already connected.")
		return nil
	}

	c := cfg.GetConfig()
	url := c.Local.Nats.Url
	user := c.Local.Nats.User
	pass := c.Local.Nats.Pass

	opts := []nats.Option{
		nats.UserInfo(user, pass),
		nats.MaxReconnects(30),
		nats.ReconnectWait(2 * time.Second),
		nats.DisconnectErrHandler(func(conn *nats.Conn, err error) {
			if err != nil {
				log.Log(log.Error, "[NATS] Disconnected: %v", err)
			} else {
				log.Log(log.Error, "[NATS] Disconnected.")
			}
		}),
		nats.ReconnectHandler(func(conn *nats.Conn) {
			log.Log(log.Info, "[NATS] Reconnected to %s", conn.ConnectedUrl())
		}),
		nats.ClosedHandler(func(conn *nats.Conn) {
			if lastErr := conn.LastError(); lastErr != nil {
				log.Log(log.Error, "[NATS] Connection closed. Reason: %v", lastErr)
			} else {
				log.Log(log.Error, "[NATS] Connection closed.")
			}
		}),
	}

	conn, err := nats.Connect(url, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	nc = conn

	log.Log(log.Info, "[NATS] Connected successfully to %s", url)
	return nil
}

func Disconnect() {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc != nil && !nc.IsClosed() {
		nc.Close()
		nc = nil
		log.Log(log.Info, "[NATS] Connection closed by user request.")
	}
}

func Publish(subject string, data []byte) error {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nats.ErrConnectionClosed
	}
	return nc.Publish(subject, data)
}

func PublishMsg(msg *nats.Msg) error {
	connectionMu.Lock()
	defer connectionMu.Unlock()

	if nc == nil || nc.IsClosed() {
		return nats.ErrConnectionClosed
	}
	return nc.PublishMsg(msg)
}

func PublishMsgWithReply(subject, reply string, data []byte) error {
	connectionMu.Lock()
	defer connectionMu.Unlock()

	if nc == nil || nc.IsClosed() {
		return nats.ErrConnectionClosed
	}
	msg := &nats.Msg{
		Subject: subject,
		Reply:   reply,
		Data:    data,
	}
	return nc.PublishMsg(msg)
}

func Subscribe(subject string, cb func(*nats.Msg)) (*nats.Subscription, error) {
	connectionMu.Lock()
	defer connectionMu.Unlock()

	if nc == nil || nc.IsClosed() {
		return nil, nats.ErrConnectionClosed
	}
	sub, err := nc.Subscribe(subject, cb)
	if err != nil {
		return nil, err
	}
	sub.SetPendingLimits(-1, -1)
	return sub, nil
}

func Request(subject string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	connectionMu.Lock()
	defer connectionMu.Unlock()

	if nc == nil || nc.IsClosed() {
		return nil, nats.ErrConnectionClosed
	}
	return nc.Request(subject, data, timeout)
}

// WaitForNodesByRole waits until we detect at least minCount nodes of a given role
// or the specified timeout elapses. Returns true if found, false otherwise.
func WaitForNodesByRole(role string, minCount int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		c := countNodesByRole(role)
		if c >= minCount {
			return true
		}
		time.Sleep(300 * time.Millisecond)
	}
	return false
}
