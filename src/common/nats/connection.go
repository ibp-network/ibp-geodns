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

// nc is the global NATS connection
var (
	nc           *nats.Conn
	connectionMu sync.Mutex
)

// Connect initializes a global NATS connection from config.json
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
		// Changed from -1 (infinite) to a finite number, e.g. 30 attempts:
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

	connection, err := nats.Connect(url, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	nc = connection

	log.Log(log.Info, "[NATS] Connected successfully to %s", url)
	return nil
}

// Disconnect closes the global NATS connection, if open
func Disconnect() {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc != nil && !nc.IsClosed() {
		nc.Close()
		nc = nil
		log.Log(log.Info, "[NATS] Connection closed by user request.")
	}
}

// Publish sends data on a subject (no reply)
func Publish(subject string, data []byte) error {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nats.ErrConnectionClosed
	}
	return nc.Publish(subject, data)
}

// PublishMsgWithReply creates and sends a message with subject, reply, and data
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

// Subscribe to a subject with a callback
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

// Request is a convenience for a single request/reply with a timeout
func Request(subject string, data []byte, timeout time.Duration) (*nats.Msg, error) {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nil, nats.ErrConnectionClosed
	}
	return nc.Request(subject, data, timeout)
}

// PublishFinalize is used by the monitor voting finalization
func PublishFinalize(msg FinalizeMessage) error {
	bytes, _ := json.Marshal(msg)
	return Publish(State.SubjectFinalize, bytes)
}
