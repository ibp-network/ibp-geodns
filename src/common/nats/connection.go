package nats

/*
   NATS connection helper – production grade.

   – Adds GetConnection() accessor for external health checks
     (required by IBPMonitor.go, issue #92).
   – All async‑callback closures defensive against nil *Conn.
*/

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

// Connect initialises the singleton NATS connection.
func Connect() error {
	connectionMu.Lock()
	defer connectionMu.Unlock()

	if nc != nil && !nc.IsClosed() {
		log.Log(log.Debug, "[NATS] Already connected")
		return nil
	}

	c := cfg.GetConfig()
	opts := []nats.Option{
		nats.UserInfo(c.Local.Nats.User, c.Local.Nats.Pass),
		nats.MaxReconnects(30),
		nats.ReconnectWait(2 * time.Second),
		nats.Timeout(10 * time.Second),

		nats.DisconnectErrHandler(func(conn *nats.Conn, err error) {
			if err != nil {
				log.Log(log.Error, "[NATS] Disconnected: %v", err)
			} else {
				log.Log(log.Error, "[NATS] Disconnected")
			}
		}),
		nats.ReconnectHandler(func(conn *nats.Conn) {
			if conn != nil {
				log.Log(log.Info, "[NATS] Re‑connected to %s", conn.ConnectedUrl())
			}
		}),
		nats.ClosedHandler(func(conn *nats.Conn) {
			if conn == nil {
				log.Log(log.Error, "[NATS] Connection closed (nil)")
				return
			}
			if last := conn.LastError(); last != nil {
				log.Log(log.Error, "[NATS] Connection closed: %v", last)
			} else {
				log.Log(log.Error, "[NATS] Connection closed")
			}
		}),
		nats.ErrorHandler(func(conn *nats.Conn, sub *nats.Subscription, err error) {
			if sub != nil {
				log.Log(log.Error, "[NATS] Async error on %s: %v", sub.Subject, err)
			} else {
				log.Log(log.Error, "[NATS] Async error: %v", err)
			}
		}),
	}

	conn, err := nats.Connect(c.Local.Nats.Url, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}
	nc = conn
	log.Log(log.Info, "[NATS] Connected to %s", c.Local.Nats.Url)
	return nil
}

// GetConnection returns the active *nats.Conn (may be nil / closed).
func GetConnection() *nats.Conn {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	return nc
}

// Disconnect forcibly closes the connection.
func Disconnect() {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc != nil && !nc.IsClosed() {
		nc.Close()
		nc = nil
		log.Log(log.Info, "[NATS] Connection closed by application")
	}
}

/*─────────────────────────────────────────────────────────────
  Publish / Subscribe helpers (unchanged)
─────────────────────────────────────────────────────────────*/

func Publish(subject string, data []byte) error {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nats.ErrConnectionClosed
	}
	if err := nc.Publish(subject, data); err != nil {
		return err
	}
	return nc.Flush()
}

func PublishMsg(msg *nats.Msg) error {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nats.ErrConnectionClosed
	}
	if err := nc.PublishMsg(msg); err != nil {
		return err
	}
	return nc.Flush()
}

func PublishMsgWithReply(subject, reply string, data []byte) error {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nats.ErrConnectionClosed
	}
	msg := &nats.Msg{Subject: subject, Reply: reply, Data: data}
	if err := nc.PublishMsg(msg); err != nil {
		return err
	}
	return nc.Flush()
}

func Subscribe(subject string, cb func(*nats.Msg)) (*nats.Subscription, error) {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nil, nats.ErrConnectionClosed
	}
	sub, err := nc.Subscribe(subject, func(m *nats.Msg) { go cb(m) })
	if err != nil {
		return nil, err
	}
	sub.SetPendingLimits(10_000, 10*1024*1024)
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
