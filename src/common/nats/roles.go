package nats

/*
   Cluster‑role management

   • JOIN is still broadcast at start‑up and on each heartbeat.
   • The former full “membership” flood (sometimes > 2 MiB) is **removed**;
     JOINs alone are fully sufficient for convergence and keep every frame
     well below the NATS default 1 MiB message limit.

   • Every outbound message is now validated to ensure Sender.NodeID != "".
*/

import (
	"encoding/json"
	"regexp"
	"strings"
	"sync/atomic"
	"time"

	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

/*───────────────────────────────────────────────────────────────────────────
  Constants & regex helpers
───────────────────────────────────────────────────────────────────────────*/

const (
	activeNodeWindow        = 5 * time.Minute
	broadcastJoinRetryCount = 3
	broadcastJoinDelay      = 500 * time.Millisecond
)

var (
	reMonitor = regexp.MustCompile(`(?i)monitor`)
	reDns     = regexp.MustCompile(`(?i)dns`)
)

/*───────────────────────────────────────────────────────────────────────────
  Atomic state (only for throttling)
───────────────────────────────────────────────────────────────────────────*/

var lastJoin int64 // unix‑nano timestamp of last JOIN we sent

/*───────────────────────────────────────────────────────────────────────────
  Public API – enable each role
───────────────────────────────────────────────────────────────────────────*/

func EnableMonitorRole() error  { return enableRoleInternal("IBPMonitor") }
func EnableDnsRole() error      { return enableRoleInternal("IBPDns") }
func EnableCollatorRole() error { return enableRoleInternal("IBPCollator") }

/*───────────────────────────────────────────────────────────────────────────
  Shared initialiser
───────────────────────────────────────────────────────────────────────────*/

func enableRoleInternal(role string) error {
	State.SubjectPropose = "consensus.propose"
	State.SubjectVote = "consensus.vote"
	State.SubjectFinalize = "consensus.finalize"
	State.SubjectCluster = "consensus.cluster"
	State.ProposalTimeout = 30 * time.Second

	if State.Proposals == nil {
		State.Proposals = make(map[ProposalID]*ProposalTracking)
	}
	if State.ClusterNodes == nil {
		State.ClusterNodes = make(map[string]NodeInfo)
	}

	State.ThisNode.NodeRole = role
	State.ThisNode.LastHeard = time.Now().UTC()

	State.Mu.Lock()
	State.ClusterNodes[State.NodeID] = State.ThisNode
	State.Mu.Unlock()

	_, err := Subscribe(">", handleAllMessages)
	if err != nil {
		return err
	}

	if role == "IBPMonitor" || role == "IBPCollator" {
		StartGarbageCollection()
	}
	startHeartbeat()

	log.Log(log.Info, "[NATS] %s role enabled for node=%s", role, State.NodeID)

	// burst a few JOINs on start‑up
	go func() {
		for i := 0; i < broadcastJoinRetryCount; i++ {
			broadcastClusterJoin()
			time.Sleep(broadcastJoinDelay)
		}
	}()

	return nil
}

/*───────────────────────────────────────────────────────────────────────────
  Heart‑beat timer – refresh LastHeard + send JOIN
───────────────────────────────────────────────────────────────────────────*/

func startHeartbeat() {
	go func() {
		time.Sleep(2 * time.Second) // let subscriptions settle
		ticker := time.NewTicker(300 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			State.Mu.Lock()
			if me, ok := State.ClusterNodes[State.NodeID]; ok {
				me.LastHeard = time.Now().UTC()
				State.ClusterNodes[State.NodeID] = me
			}
			State.Mu.Unlock()
			broadcastClusterJoin()
		}
	}()
}

/*───────────────────────────────────────────────────────────────────────────
  JOIN broadcaster (membership flood removed)
───────────────────────────────────────────────────────────────────────────*/

func broadcastClusterJoin() {
	now := time.Now().UnixNano()
	// throttle JOINs to once every 5 s except for the initial burst
	if last := atomic.LoadInt64(&lastJoin); last != 0 && now-last < 5*int64(time.Second) {
		return
	}
	atomic.StoreInt64(&lastJoin, now)

	if State.ThisNode.NodeID == "" {
		log.Log(log.Error, "[NATS] broadcastClusterJoin: NodeID is empty – skipping")
		return
	}
	msg := ClusterMessage{
		Type:   "join",
		Sender: State.ThisNode,
	}
	data, _ := json.Marshal(msg)
	if err := Publish(State.SubjectCluster, data); err != nil {
		log.Log(log.Error, "[NATS] Failed to publish JOIN: %v", err)
	}
}

/*───────────────────────────────────────────────────────────────────────────
  Wildcard dispatcher
───────────────────────────────────────────────────────────────────────────*/

func handleAllMessages(m *nats.Msg) {
	go func() {
		subj := m.Subject
		if subj == State.SubjectCluster {
			handleClusterMessage(m)
			return
		}

		switch State.ThisNode.NodeRole {
		case "IBPMonitor":
			switch subj {
			case State.SubjectPropose:
				handleProposal(m)
			case State.SubjectVote:
				handleVote(m)
			case State.SubjectFinalize:
				handleFinalize(m)
			case "monitor.stats.getDowntime":
				handleMonitorStatsRequest(m)
			default:
				if strings.Contains(subj, "downtimeReply") {
					handleMonitorStatsData(m)
				}
			}

		case "IBPDns":
			if subj == "dns.usage.getUsage" {
				handleDnsUsageRequest(m)
			} else if strings.Contains(subj, "usageReply") {
				handleDnsUsageData(m)
			}

		case "IBPCollator":
			switch {
			case subj == "monitor.stats.downtimeData" || strings.Contains(subj, "downtimeReply"):
				handleMonitorStatsData(m)
			case subj == "dns.usage.usageData" || strings.Contains(subj, "usageReply"):
				handleDnsUsageData(m)
			}
		}
	}()
}

/*───────────────────────────────────────────────────────────────────────────
  Cluster message processing (JOIN only)
───────────────────────────────────────────────────────────────────────────*/

func handleClusterMessage(m *nats.Msg) {
	var msg ClusterMessage
	if err := json.Unmarshal(m.Data, &msg); err != nil {
		log.Log(log.Error, "[NATS] handleClusterMessage: unmarshal error: %v", err)
		return
	}

	if msg.Sender.NodeID == "" {
		// Silently discard – in practice this only happens with malformed
		// legacy frames still present on the server.
		return
	}

	markNodeHeard(msg.Sender.NodeID)

	if msg.Type == "join" {
		addNode(msg.Sender)
	}
}

/*───────────────────────────────────────────────────────────────────────────
  Node bookkeeping helpers
───────────────────────────────────────────────────────────────────────────*/

func addNode(n NodeInfo) {
	State.Mu.Lock()
	defer State.Mu.Unlock()

	if n.NodeID == "" {
		return
	}
	cur, exists := State.ClusterNodes[n.NodeID]
	if !exists || (cur.NodeRole == "" && n.NodeRole != "") {
		State.ClusterNodes[n.NodeID] = n
	}
}

func markNodeHeard(id string) {
	if id == "" {
		return
	}
	State.Mu.Lock()
	defer State.Mu.Unlock()

	n, exists := State.ClusterNodes[id]
	if !exists {
		n = NodeInfo{NodeID: id}
	}
	if n.NodeRole == "" {
		n.NodeRole = guessRoleFromID(id)
	}
	n.LastHeard = time.Now().UTC()
	State.ClusterNodes[id] = n
}

func guessRoleFromID(id string) string {
	switch {
	case reMonitor.MatchString(id):
		return "IBPMonitor"
	case reDns.MatchString(id):
		return "IBPDns"
	default:
		return ""
	}
}

/*───────────────────────────────────────────────────────────────────────────
  Liveness helpers (unchanged logic)
───────────────────────────────────────────────────────────────────────────*/

func IsNodeActive(n NodeInfo) bool {
	return n.NodeID != "" && !n.LastHeard.IsZero() && time.Since(n.LastHeard) < activeNodeWindow
}

func CountActiveMonitors() int {
	State.Mu.RLock()
	defer State.Mu.RUnlock()
	cnt := 0
	for _, n := range State.ClusterNodes {
		if n.NodeRole == "IBPMonitor" && IsNodeActive(n) {
			cnt++
		}
	}
	return cnt
}

func CountActiveDns() int {
	State.Mu.RLock()
	defer State.Mu.RUnlock()
	cnt := 0
	for _, n := range State.ClusterNodes {
		if n.NodeRole == "IBPDns" && IsNodeActive(n) {
			cnt++
		}
	}
	return cnt
}

/*───────────────────────────────────────────────────────────────────────────
  Garbage‑collection (unchanged)
───────────────────────────────────────────────────────────────────────────*/

func StartGarbageCollection() {
	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for range t.C {
			cleanOldProposals()
			cleanStaleNodes()
		}
	}()
}

func cleanOldProposals() {
	State.Mu.Lock()
	defer State.Mu.Unlock()

	now := time.Now().UTC()
	for id, pt := range State.Proposals {
		if now.Sub(pt.Proposal.Timestamp) > 10*time.Minute {
			if pt.Timer != nil {
				pt.Timer.Stop()
			}
			delete(State.Proposals, id)
		}
	}
}

func cleanStaleNodes() {
	now := time.Now().UTC()
	State.Mu.Lock()
	defer State.Mu.Unlock()

	for id, n := range State.ClusterNodes {
		if id == State.NodeID {
			continue
		}
		if !n.LastHeard.IsZero() && now.Sub(n.LastHeard) > 15*time.Minute {
			delete(State.ClusterNodes, id)
		}
	}
}

/*───────────────────────────────────────────────────────────────────────────
  Export package‑internal helpers
───────────────────────────────────────────────────────────────────────────*/

var (
	countActiveMonitors = CountActiveMonitors
	countActiveDns      = CountActiveDns
	isNodeActive        = IsNodeActive
)
