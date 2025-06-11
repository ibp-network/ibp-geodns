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

// GetConnection exposes the live *nats.Conn so other packages (consensus
// manager, etc.) can publish/subscribe without creating a second socket.
func GetConnection() *nats.Conn {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	return nc
}

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
		nats.Timeout(10 * time.Second),
		nats.PingInterval(20 * time.Second),
		nats.MaxPingsOutstanding(5),
		nats.DisconnectErrHandler(func(conn *nats.Conn, err error) {
			if err != nil {
				log.Log(log.Error, "[NATS] Disconnected: %v", err)
			} else {
				log.Log(log.Error, "[NATS] Disconnected.")
			}
		}),
		nats.ReconnectHandler(func(conn *nats.Conn) {
			log.Log(log.Info, "[NATS] Re‑connected to %s", conn.ConnectedUrl())
		}),
		nats.ClosedHandler(func(conn *nats.Conn) {
			if lastErr := conn.LastError(); lastErr != nil {
				log.Log(log.Error, "[NATS] Connection closed. Reason: %v", lastErr)
			} else {
				log.Log(log.Error, "[NATS] Connection closed.")
			}
		}),
		nats.ErrorHandler(func(conn *nats.Conn, sub *nats.Subscription, err error) {
			log.Log(log.Error, "[NATS] Async error: sub=%v err=%v", sub.Subject, err)
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

// Publish publishes to subject
func Publish(subject string, data []byte) error {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nats.ErrConnectionClosed
	}
	err := nc.Publish(subject, data)
	if err != nil {
		return err
	}
	// Flush to ensure message is sent immediately
	return nc.Flush()
}

// PublishMsg is wrapper
func PublishMsg(msg *nats.Msg) error {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nats.ErrConnectionClosed
	}
	err := nc.PublishMsg(msg)
	if err != nil {
		return err
	}
	// Flush to ensure message is sent immediately
	return nc.Flush()
}

// PublishMsgWithReply publishes with a reply subject
func PublishMsgWithReply(subject, reply string, data []byte) error {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nats.ErrConnectionClosed
	}
	msg := &nats.Msg{Subject: subject, Reply: reply, Data: data}
	err := nc.PublishMsg(msg)
	if err != nil {
		return err
	}
	// Flush to ensure message is sent immediately
	return nc.Flush()
}

// Subscribe to a subject
func Subscribe(subject string, cb func(*nats.Msg)) (*nats.Subscription, error) {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nil, nats.ErrConnectionClosed
	}

	sub, err := nc.Subscribe(subject, func(msg *nats.Msg) {
		// Process callback in a goroutine to prevent blocking
		go cb(msg)
	})
	if err != nil {
		return nil, err
	}
	// Set reasonable limits instead of unlimited
	sub.SetPendingLimits(10000, 10*1024*1024) // 10k messages or 10MB
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
