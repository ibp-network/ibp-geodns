package nats

import (
	"fmt"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

// Connect tries to establish a new NATS connection with automatic reconnection
// and appropriate event handlers. If successful, it stores the connection in 'nc'.
func Connect() error {
	c := cfg.GetConfig()

	natsMu.Lock()
	defer natsMu.Unlock()

	if nc != nil && !nc.IsClosed() {
		// Already connected
		return nil
	}

	opts := []nats.Option{
		nats.UserInfo(c.Local.Nats.User, c.Local.Nats.Pass),
		nats.MaxReconnects(-1), // infinite reconnect attempts
		nats.ReconnectWait(2 * time.Second),
		nats.DisconnectErrHandler(func(conn *nats.Conn, err error) {
			if err != nil {
				log.Log(log.Error, "[NATS] Disconnected due to: %v", err)
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

	var err error
	nc, err = nats.Connect(c.Local.Nats.Url, opts...)
	if err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	log.Log(log.Info, "[NATS] Connected successfully to %s", c.Local.Nats.Url)
	return nil
}
