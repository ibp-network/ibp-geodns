package nats

import (
	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// We store global state about the node, proposals, etc.
var (
	nc     *nats.Conn
	natsMu sync.Mutex
	state  NodeState
)

// Init sets up the NATS connection, subscribes to relevant subjects,
// and starts a garbage-collection plus connection monitor goroutine.
func Init() {
	c := cfg.GetConfig()
	con := c.Local.Nats

	state.NodeID = con.NodeID
	state.ThisNode = NodeInfo{
		NodeID: con.NodeID,
	}

	state.Proposals = make(map[ProposalID]*ProposalTracking)
	state.ClusterNodes = make(map[string]NodeInfo)
	state.ClusterNodes[state.NodeID] = state.ThisNode

	state.SubjectPropose = "consensus.propose"
	state.SubjectVote = "consensus.vote"
	state.SubjectFinalize = "consensus.finalize"
	state.SubjectCluster = "consensus.cluster"

	// We set a default proposal timeout
	state.ProposalTimeout = 12 * time.Second
	state.NatsUrl = con.Url

	// Attempt initial connection
	err := Connect()
	if err != nil {
		log.Log(log.Fatal, "Nats connection error: %+v", err)
	}

	// Subscribe to relevant topics
	err = Subscribe()
	if err != nil {
		log.Log(log.Fatal, "Nats subscription error: %+v", err)
	}

	// Start garbage collection of proposals
	go StartGarbageCollection()

	// Start a connection monitor
	startConnectionMonitor()
}

// startConnectionMonitor periodically logs the current NATS connection status.
// Because we configure NATS to automatically reconnect, this mostly helps us
// debug or detect if the connection is stuck or closed.
func startConnectionMonitor() {
	go func() {
		for {
			time.Sleep(10 * time.Second)
			natsMu.Lock()
			if nc == nil {
				log.Log(log.Warn, "[NATS] Connection handle is nil.")
				natsMu.Unlock()
				continue
			}
			status := nc.Status()
			natsMu.Unlock()

			switch status {
			case nats.CONNECTED:
				// Debug logging to confirm we remain connected
				log.Log(log.Debug, "[NATS] Status: CONNECTED")
			case nats.RECONNECTING:
				log.Log(log.Warn, "[NATS] Status: RECONNECTING...")
			case nats.CLOSED:
				log.Log(log.Error, "[NATS] Status: CLOSED")
			default:
				log.Log(log.Warn, "[NATS] Status: %v", status)
			}
		}
	}()
}
