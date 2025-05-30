package nats

import (
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
)

var (
	nc           *nats.Conn
	connectionMu sync.Mutex
)

// Connect initializes a global NATS connection from the config.json data.
func Connect() error {
	connectionMu.Lock()
	defer connectionMu.Unlock()

	// If we already have a live connection, skip
	if nc != nil && !nc.IsClosed() {
		log.Log(log.Debug, "[NATS] Already connected.")
		return nil
	}

	// Pull from your config package
	c := cfg.GetConfig()
	url := c.Local.Nats.Url
	user := c.Local.Nats.User
	pass := c.Local.Nats.Pass

	// Build NATS options
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

	// Attempt connection
	conn, err := nats.Connect(url, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	nc = conn

	log.Log(log.Info, "[NATS] Connected successfully to %s", url)
	return nil
}

// Disconnect closes the global NATS connection, if open.
func Disconnect() {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc != nil && !nc.IsClosed() {
		nc.Close()
		nc = nil
		log.Log(log.Info, "[NATS] Connection closed by user request.")
	}
}

// Publish sends data to a subject (fire-and-forget).
func Publish(subject string, data []byte) error {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nats.ErrConnectionClosed
	}
	return nc.Publish(subject, data)
}

// PublishMsg sends a raw nats.Msg (if you want to set .Reply or .Header).
func PublishMsg(msg *nats.Msg) error {
	connectionMu.Lock()
	defer connectionMu.Unlock()

	if nc == nil || nc.IsClosed() {
		return nats.ErrConnectionClosed
	}
	return nc.PublishMsg(msg)
}

// PublishMsgWithReply builds a nats.Msg with subject+reply+data, then publishes it.
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

// Subscribe creates a subscription to a subject with a callback.
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

// Request is a convenience for a single request/reply with timeout.
func Request(subject string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	connectionMu.Lock()
	defer connectionMu.Unlock()

	if nc == nil || nc.IsClosed() {
		return nil, nats.ErrConnectionClosed
	}
	return nc.Request(subject, data, timeout)
}
