package nats

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
  Constants and helpers
───────────────────────────────────────────────────────────────────────────*/

const (
	activeNodeWindow        = 5 * time.Minute
	membershipMinInterval   = 30 * time.Second // throttle cluster floods
	broadcastJoinRetryCount = 3
	broadcastJoinRetryDelay = 500 * time.Millisecond
)

// simple regex‑based role inference when a node never sent a “join”
var (
	reMonitor = regexp.MustCompile(`(?i)monitor`)
	reDns     = regexp.MustCompile(`(?i)dns`)
)

/*───────────────────────────────────────────────────────────────────────────
  Internal globals
───────────────────────────────────────────────────────────────────────────*/

var lastMembershipBroadcast int64 // unix‑nano, atomically accessed

/*───────────────────────────────────────────────────────────────────────────
  Public role‑enable entry‑points
───────────────────────────────────────────────────────────────────────────*/

func EnableMonitorRole() error  { return enableRoleInternal("IBPMonitor") }
func EnableDnsRole() error      { return enableRoleInternal("IBPDns") }
func EnableCollatorRole() error { return enableRoleInternal("IBPCollator") }

/*───────────────────────────────────────────────────────────────────────────
  Shared initialiser for each role
───────────────────────────────────────────────────────────────────────────*/

func enableRoleInternal(role string) error {
	State.SubjectPropose = "consensus.propose"
	State.SubjectVote = "consensus.vote"
	State.SubjectFinalize = "consensus.finalize"
	State.SubjectCluster = "consensus.cluster"
	State.ProposalTimeout = 30 * time.Second // monitors only; harmless for others

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

	// wildcard subscription → one central dispatcher
	if _, err := Subscribe(">", handleAllMessages); err != nil {
		return err
	}

	if role == "IBPMonitor" || role == "IBPCollator" {
		StartGarbageCollection()
	}
	startHeartbeat()

	log.Log(log.Info, "[NATS] %s role enabled for node=%s", role, State.NodeID)

	// burst a few join packets so every peer learns us quickly
	go func() {
		for i := 0; i < broadcastJoinRetryCount; i++ {
			broadcastClusterJoin()
			time.Sleep(broadcastJoinRetryDelay)
		}
	}()

	return nil
}

/*───────────────────────────────────────────────────────────────────────────
  Heart‑beat
───────────────────────────────────────────────────────────────────────────*/

func startHeartbeat() {
	go func() {
		time.Sleep(2 * time.Second) // let subscriptions settle
		ticker := time.NewTicker(300 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			State.Mu.Lock()
			if node, ok := State.ClusterNodes[State.NodeID]; ok {
				node.LastHeard = time.Now().UTC()
				State.ClusterNodes[State.NodeID] = node
			}
			State.Mu.Unlock()
			broadcastClusterJoin()
		}
	}()
}

/*───────────────────────────────────────────────────────────────────────────
  Cluster JOIN & MEMBERSHIP broadcasters
───────────────────────────────────────────────────────────────────────────*/

func broadcastClusterJoin() {
	if State.ThisNode.NodeID == "" || State.ThisNode.NodeRole == "" {
		log.Log(log.Error, "[NATS] broadcastClusterJoin: ThisNode incomplete (id=%s role=%s)",
			State.ThisNode.NodeID, State.ThisNode.NodeRole)
		return
	}

	msg := ClusterMessage{
		Type:   "join",
		Sender: State.ThisNode,
	}
	data, _ := json.Marshal(msg)
	if err := Publish(State.SubjectCluster, data); err != nil {
		log.Log(log.Error, "[NATS] Failed to publish cluster join: %v", err)
		return
	}
	log.Log(log.Debug, "[NATS] broadcastClusterJoin → %s", State.SubjectCluster)
}

func broadcastClusterMembership() {
	now := time.Now().UnixNano()
	if atomic.LoadInt64(&lastMembershipBroadcast) != 0 &&
		now-atomic.LoadInt64(&lastMembershipBroadcast) < membershipMinInterval.Nanoseconds() {
		return // skip – still inside throttle window
	}

	State.Mu.RLock()
	all := make([]NodeInfo, 0, len(State.ClusterNodes))
	for _, n := range State.ClusterNodes {
		all = append(all, n)
	}
	State.Mu.RUnlock()

	msg := ClusterMessage{
		Type:    "membership",
		Sender:  State.ThisNode,
		Members: all,
	}
	data, _ := json.Marshal(msg)
	if err := Publish(State.SubjectCluster, data); err != nil {
		log.Log(log.Error, "[NATS] Failed to broadcast membership: %v", err)
		return
	}

	atomic.StoreInt64(&lastMembershipBroadcast, now)
	log.Log(log.Debug, "[NATS] broadcastClusterMembership: %d nodes (%d bytes)", len(all), len(data))
}

/*───────────────────────────────────────────────────────────────────────────
  Wildcard dispatcher – routes based on role
───────────────────────────────────────────────────────────────────────────*/

func handleAllMessages(m *nats.Msg) {
	go func() {
		subj := m.Subject

		// always process cluster messages
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
  Cluster message handler – now robust to missing NodeRole
───────────────────────────────────────────────────────────────────────────*/

func handleClusterMessage(m *nats.Msg) {
	var cm ClusterMessage
	if err := json.Unmarshal(m.Data, &cm); err != nil {
		log.Log(log.Error, "[NATS] handleClusterMessage: unmarshal error: %v", err)
		return
	}

	if cm.Sender.NodeID == "" {
		log.Log(log.Error, "[NATS] handleClusterMessage: received message with empty NodeID")
		return
	}
	markNodeHeard(cm.Sender.NodeID)

	switch cm.Type {
	case "join":
		addNode(cm.Sender)
		broadcastClusterMembership() // respond with full list
	case "membership":
		mergeClusterMembership(cm.Members)
	default:
		log.Log(log.Warn, "[NATS] handleClusterMessage: unknown type=%s", cm.Type)
	}
}

/*───────────────────────────────────────────────────────────────────────────
  Membership‑merge helpers (UNCHANGED except tiny tweak)
───────────────────────────────────────────────────────────────────────────*/

func mergeClusterMembership(in []NodeInfo) {
	State.Mu.Lock()
	defer State.Mu.Unlock()

	for _, n := range in {
		if n.NodeID == "" {
			continue
		}
		cur, exists := State.ClusterNodes[n.NodeID]
		// keep whichever has a non‑empty role
		if !exists || (cur.NodeRole == "" && n.NodeRole != "") {
			State.ClusterNodes[n.NodeID] = n
		}
	}
}

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

/*───────────────────────────────────────────────────────────────────────────
  Role inference + LastHeard update
───────────────────────────────────────────────────────────────────────────*/

func guessRoleFromNodeID(id string) string {
	switch {
	case reMonitor.MatchString(id):
		return "IBPMonitor"
	case reDns.MatchString(id):
		return "IBPDns"
	default:
		return ""
	}
}

func markNodeHeard(nodeID string) {
	if nodeID == "" {
		return
	}
	State.Mu.Lock()
	defer State.Mu.Unlock()

	n, exists := State.ClusterNodes[nodeID]
	if !exists {
		n = NodeInfo{NodeID: nodeID}
	}
	if n.NodeRole == "" {
		n.NodeRole = guessRoleFromNodeID(nodeID)
	}
	n.LastHeard = time.Now().UTC()
	State.ClusterNodes[nodeID] = n
}

/*───────────────────────────────────────────────────────────────────────────
  Garbage‑collection & liveness (UNCHANGED)
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
	for pid, pt := range State.Proposals {
		if now.Sub(pt.Proposal.Timestamp) > 10*time.Minute {
			if pt.Timer != nil {
				pt.Timer.Stop()
			}
			delete(State.Proposals, pid)
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
  Public liveness helpers (UNCHANGED)
───────────────────────────────────────────────────────────────────────────*/

func IsNodeActive(n NodeInfo) bool {
	if n.NodeID == "" || n.LastHeard.IsZero() {
		return false
	}
	return time.Since(n.LastHeard) < activeNodeWindow
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
  Export un‑capitalised helpers for other files in package
───────────────────────────────────────────────────────────────────────────*/

var (
	countActiveMonitors = CountActiveMonitors
	countActiveDns      = CountActiveDns
	isNodeActive        = IsNodeActive
)
