package nats

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

var (
	connectionMu sync.Mutex
	nc           *nats.Conn
)

// Connect initializes a global NATS connection from the project's config.
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
		nats.MaxReconnects(-1),
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

	connection, err := nats.Connect(url, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	nc = connection
	log.Log(log.Info, "[NATS] Connected successfully to %s", url)
	return nil
}

// Disconnect closes the global NATS connection (if open).
func Disconnect() {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc != nil && !nc.IsClosed() {
		nc.Close()
		nc = nil
		log.Log(log.Info, "[NATS] Connection closed by user request.")
	}
}

// Publish sends a message to a subject without a reply.
func Publish(subject string, data []byte) error {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nats.ErrConnectionClosed
	}
	return nc.Publish(subject, data)
}

// PublishMsgWithReply creates and sends a message with a subject, a reply subject, and data.
// This is used by Collator to publish a request with a specified reply subject (inbox).
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

// Subscribe registers a callback for the given subject.
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
	// Large pending limits for reliability
	sub.SetPendingLimits(-1, -1)
	return sub, nil
}

// Request sends a request and waits for a single reply (typical request/response).
func Request(subject string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nil, nats.ErrConnectionClosed
	}
	return nc.Request(subject, data, timeout)
}

// PublishFinalize is a helper for the consensus finalization broadcast.
func PublishFinalize(msg FinalizeMessage) error {
	bytes, _ := json.Marshal(msg)
	return Publish(State.SubjectFinalize, bytes)
}
