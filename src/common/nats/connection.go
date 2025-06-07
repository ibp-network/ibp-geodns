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

// Connect sets up the global NATS connection
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

// Disconnect forcibly closes
func Disconnect() {
	connectionMu.Lock()
	defer connectionMu.Unlock()

	if nc != nil && !nc.IsClosed() {
		nc.Close()
		nc = nil
		log.Log(log.Info, "[NATS] Connection closed by user request.")
	}
}

// Publish publishes to subject
func Publish(subject string, data []byte) error {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nats.ErrConnectionClosed
	}
	return nc.Publish(subject, data)
}

// PublishMsg is wrapper
func PublishMsg(msg *nats.Msg) error {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nats.ErrConnectionClosed
	}
	return nc.PublishMsg(msg)
}

// PublishMsgWithReply publishes with a reply subject
func PublishMsgWithReply(subject, reply string, data []byte) error {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nats.ErrConnectionClosed
	}
	msg := &nats.Msg{Subject: subject, Reply: reply, Data: data}
	return nc.PublishMsg(msg)
}

// Subscribe to a subject
func Subscribe(subject string, cb func(*nats.Msg)) (*nats.Subscription, error) {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nil, nats.ErrConnectionClosed
	}

	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		// The callback is invoked asynchronously by the nats library
		cb(msg)
	})
	if err != nil {
		return nil, err
	}
	sub.SetPendingLimits(-1, -1)
	return sub, nil
}

// Request is optional
func Request(subject string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nil, nats.ErrConnectionClosed
	}
	return nc.Request(subject, data, timeout)
}

// Flush ensures that all subscriptions and messages are processed up to this point.
// This helps to avoid losing messages if we publish too quickly after subscribing.
func Flush() error {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nats.ErrConnectionClosed
	}
	return nc.Flush()
}
