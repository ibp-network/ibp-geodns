package nats

import (
	"fmt"
	"strings"
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

func GetConnection() *nats.Conn {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	return nc
}

func Connect() error {
	connectionMu.Lock()
	defer connectionMu.Unlock()

	if nc != nil && !nc.IsClosed() {
		return nil
	}

	c := cfg.GetConfig()
	opts := []nats.Option{
		nats.UserInfo(c.Local.Nats.User, c.Local.Nats.Pass),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2 * time.Second),
		nats.Timeout(10 * time.Second),
		nats.PingInterval(200 * time.Second),
		nats.MaxPingsOutstanding(5),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			log.Log(log.Error, "[NATS] Disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(conn *nats.Conn) {
			log.Log(log.Info, "[NATS] Re‑connected to %s", conn.ConnectedUrl())
		}),
		nats.ClosedHandler(func(conn *nats.Conn) {
			if e := conn.LastError(); e != nil {
				log.Log(log.Error, "[NATS] Connection closed: %v", e)
			}
		}),
		nats.ErrorHandler(func(conn *nats.Conn, sub *nats.Subscription, err error) {
			if err != nil && (strings.Contains(err.Error(), "wsasend") ||
				strings.Contains(err.Error(), "wsarecv")) {
				log.Log(log.Debug, "[NATS] Async I/O reset: %v", err)
			} else if err != nil {
				if sub != nil {
					log.Log(log.Error, "[NATS] Async error on %s: %v", sub.Subject, err)
				} else {
					log.Log(log.Error, "[NATS] Async error: %v", err)
				}
			}
		}),
	}

	conn, err := nats.Connect(c.Local.Nats.Url, opts...)
	if err != nil {
		return fmt.Errorf("failed NATS connect: %w", err)
	}
	nc = conn
	log.Log(log.Info, "[NATS] Connected to %s", conn.ConnectedUrl())
	return nil
}

func Disconnect() {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc != nil && !nc.IsClosed() {
		nc.Close()
		nc = nil
	}
}

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

func PublishMsgWithReply(subject, reply string, data []byte) error {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc == nil || nc.IsClosed() {
		return nats.ErrConnectionClosed
	}
	if err := nc.PublishMsg(&nats.Msg{Subject: subject, Reply: reply, Data: data}); err != nil {
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
	sub.SetPendingLimits(1000000, 128000000)
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
