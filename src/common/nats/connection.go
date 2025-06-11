package nats

/*
   NATS connection helper  – production grade.

   ▪ Adds GetConnection() accessor      (required by IBPMonitor)
   ▪ Robust reconnect strategy:
       – reconnect‑buffer 32 MiB
       – exponential back‑off (100 ms … 4 s) + jitter
       – tighter ping / outstanding ping settings
   ▪ Downgrades the common “wsasend/wsarecv: connection forcibly closed”
     noise from ERROR → DEBUG so logs stay clean while still surfacing
     unexpected problems.
*/

import (
	"fmt"
	"math/rand"
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

/*─────────────────────────────────────────────────────────────
  Connect / Disconnect
─────────────────────────────────────────────────────────────*/

// Connect initialises the singleton NATS connection.
func Connect() error {
	connectionMu.Lock()
	defer connectionMu.Unlock()

	if nc != nil && !nc.IsClosed() {
		return nil
	}

	c := cfg.GetConfig()

	opts := buildOptions(c.Local.Nats.User, c.Local.Nats.Pass)

	conn, err := nats.Connect(c.Local.Nats.Url, opts...)
	if err != nil {
		return fmt.Errorf("NATS connect error: %w", err)
	}
	nc = conn
	log.Log(log.Info, "[NATS] Connected to %s", conn.ConnectedUrl())
	return nil
}

// Disconnect closes the current connection (if any).
func Disconnect() {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	if nc != nil && !nc.IsClosed() {
		nc.Drain()
		nc.Close()
		nc = nil
		log.Log(log.Info, "[NATS] Connection closed by application")
	}
}

// GetConnection returns the active *nats.Conn (nil if none/closed).
func GetConnection() *nats.Conn {
	connectionMu.Lock()
	defer connectionMu.Unlock()
	return nc
}

/*─────────────────────────────────────────────────────────────
  Option builder – central place to keep behaviour consistent
─────────────────────────────────────────────────────────────*/

func buildOptions(user, pass string) []nats.Option {
	var opts []nats.Option

	/* authentication -------------------------------------------------------*/
	opts = append(opts, nats.UserInfo(user, pass))

	/* reconnect strategy ---------------------------------------------------*/
	opts = append(opts,
		nats.MaxReconnects(-1),              //   infinite
		nats.ReconnectBufSize(32*1024*1024), //   32 MiB buffer
		nats.CustomReconnectDelay(func(attempt int) time.Duration {
			// exponential back‑off (100 ms → 4 s) + jitter ±50 ms
			base := time.Duration(100*(1<<uint(min(attempt, 5)))) * time.Millisecond
			jitter := time.Duration(rand.Intn(100)-50) * time.Millisecond
			return base + jitter
		}),
	)

	/* low‑latency liveness --------------------------------------------------*/
	opts = append(opts,
		nats.PingInterval(20*time.Second),
		nats.MaxPingsOutstanding(2),
		nats.Timeout(10*time.Second),
	)

	/* async callbacks ------------------------------------------------------*/
	opts = append(opts,
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
			// The forcibly‑closed‑by‑remote message is expected during LB rotation;
			// log it at DEBUG to avoid noise. Everything else remains an error.
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
	)

	return opts
}

/*─────────────────────────────────────────────────────────────
  Publish / Request / Subscribe – unchanged
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

/*─────────────────────────────────────────────────────────────
  Helpers
─────────────────────────────────────────────────────────────*/

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
