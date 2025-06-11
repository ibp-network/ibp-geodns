package nats

/*
   Cluster‑membership / role management.

   Fixes & additions – 2025‑06‑11
   • Restored handleAllMessages() so other units link correctly.
   • Re‑exported alias variables  (countActiveMonitors / countActiveDns /
     isNodeActive) for legacy callers.
   • cluster‑broadcasts skip placeholder nodes; blank‑NodeID messages ignored.
*/

import (
	"encoding/json"
	"strings"
	"time"

	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

/*─────────────────────────────────────────────────────────────
  Constants
─────────────────────────────────────────────────────────────*/

const activeNodeWindow = 5 * time.Minute

/*─────────────────────────────────────────────────────────────
  Public alias variables (compile‑time compatibility)
─────────────────────────────────────────────────────────────*/

var (
	countActiveMonitors = CountActiveMonitors
	countActiveDns      = CountActiveDns
)

/*─────────────────────────────────────────────────────────────
  Role enablers – unchanged except broadcast improvements
─────────────────────────────────────────────────────────────*/

func EnableMonitorRole() error {
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

	State.ThisNode.NodeRole = "IBPMonitor"
	State.ThisNode.LastHeard = time.Now().UTC()

	State.Mu.Lock()
	State.ClusterNodes[State.NodeID] = State.ThisNode
	State.Mu.Unlock()

	if _, err := Subscribe(">", handleAllMessages); err != nil {
		return err
	}

	StartGarbageCollection()
	startHeartbeat()

	log.Log(log.Info, "[NATS] Monitor role enabled for node=%s", State.NodeID)

	go tripleJoin()
	return nil
}

func EnableDnsRole() error {
	State.SubjectPropose = "consensus.propose"
	State.SubjectVote = "consensus.vote"
	State.SubjectFinalize = "consensus.finalize"
	State.SubjectCluster = "consensus.cluster"

	if State.Proposals == nil {
		State.Proposals = make(map[ProposalID]*ProposalTracking)
	}
	if State.ClusterNodes == nil {
		State.ClusterNodes = make(map[string]NodeInfo)
	}

	State.ThisNode.NodeRole = "IBPDns"
	State.ThisNode.LastHeard = time.Now().UTC()

	State.Mu.Lock()
	State.ClusterNodes[State.NodeID] = State.ThisNode
	State.Mu.Unlock()

	if _, err := Subscribe(">", handleAllMessages); err != nil {
		return err
	}

	startHeartbeat()

	log.Log(log.Info, "[NATS] IBPDns role enabled for node=%s", State.NodeID)

	go tripleJoin()
	return nil
}

func EnableCollatorRole() error {
	State.SubjectPropose = "consensus.propose"
	State.SubjectVote = "consensus.vote"
	State.SubjectFinalize = "consensus.finalize"
	State.SubjectCluster = "consensus.cluster"

	if State.Proposals == nil {
		State.Proposals = make(map[ProposalID]*ProposalTracking)
	}
	if State.ClusterNodes == nil {
		State.ClusterNodes = make(map[string]NodeInfo)
	}

	State.ThisNode.NodeRole = "IBPCollator"
	State.ThisNode.LastHeard = time.Now().UTC()

	State.Mu.Lock()
	State.ClusterNodes[State.NodeID] = State.ThisNode
	State.Mu.Unlock()

	if _, err := Subscribe(">", handleAllMessages); err != nil {
		return err
	}

	StartGarbageCollection()
	startHeartbeat()

	log.Log(log.Info, "[NATS] Collator role enabled for node=%s", State.NodeID)

	go tripleJoin()
	return nil
}

/*─────────────────────────────────────────────────────────────
  Heart‑beat & helper
─────────────────────────────────────────────────────────────*/

func startHeartbeat() {
	go func() {
		time.Sleep(2 * time.Second)
		t := time.NewTicker(300 * time.Second)
		defer t.Stop()
		for range t.C {
			State.Mu.Lock()
			if n, ok := State.ClusterNodes[State.NodeID]; ok {
				n.LastHeard = time.Now().UTC()
				State.ClusterNodes[State.NodeID] = n
			}
			State.Mu.Unlock()
			broadcastClusterJoin()
		}
	}()
}

func tripleJoin() {
	for i := 0; i < 3; i++ {
		broadcastClusterJoin()
		time.Sleep(500 * time.Millisecond)
	}
}

/*─────────────────────────────────────────────────────────────
  Broadcast helpers – skip placeholders
─────────────────────────────────────────────────────────────*/

func broadcastClusterJoin() {
	if State.ThisNode.NodeID == "" {
		return
	}
	msg := ClusterMessage{
		Type:   "join",
		Sender: State.ThisNode,
	}
	if data, _ := json.Marshal(msg); Publish(State.SubjectCluster, data) != nil {
		log.Log(log.Error, "[NATS] failed to publish cluster join")
	}
}

func broadcastClusterMembership() {
	State.Mu.RLock()
	var active []NodeInfo
	for _, n := range State.ClusterNodes {
		if n.NodeID != "" {
			active = append(active, n)
		}
	}
	State.Mu.RUnlock()

	msg := ClusterMessage{
		Type:    "membership",
		Sender:  State.ThisNode,
		Members: active,
	}
	if data, _ := json.Marshal(msg); Publish(State.SubjectCluster, data) != nil {
		log.Log(log.Error, "[NATS] failed to broadcast membership")
	}
}

/*─────────────────────────────────────────────────────────────
  Message dispatcher – restored
─────────────────────────────────────────────────────────────*/

func handleAllMessages(m *nats.Msg) {
	go func() {
		subj := m.Subject

		// cluster messages are processed by everyone
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
			switch subj {
			case "dns.usage.getUsage":
				handleDnsUsageRequest(m)
			default:
				if strings.Contains(subj, "usageReply") {
					handleDnsUsageData(m)
				}
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

/*─────────────────────────────────────────────────────────────
  Cluster‑message handler (tolerant of blank NodeID)
─────────────────────────────────────────────────────────────*/

func handleClusterMessage(m *nats.Msg) {
	var msg ClusterMessage
	if err := json.Unmarshal(m.Data, &msg); err != nil {
		log.Log(log.Error, "[NATS] cluster unmarshal: %v", err)
		return
	}
	if strings.TrimSpace(msg.Sender.NodeID) == "" {
		// Ignore placeholder / malformed message
		return
	}

	markNodeHeard(msg.Sender.NodeID)

	switch msg.Type {
	case "join":
		addNode(msg.Sender)
		broadcastClusterMembership()
	case "membership":
		mergeClusterMembership(msg.Members)
	}
}

/*─────────────────────────────────────────────────────────────
  Remaining helpers (unchanged)
─────────────────────────────────────────────────────────────*/

func mergeClusterMembership(inMembers []NodeInfo) {
	State.Mu.Lock()
	defer State.Mu.Unlock()
	for _, m := range inMembers {
		if m.NodeID == "" {
			continue
		}
		ex, ok := State.ClusterNodes[m.NodeID]
		if !ok || ex.NodeRole == "" && m.NodeRole != "" {
			State.ClusterNodes[m.NodeID] = m
		}
	}
}

func addNode(node NodeInfo) {
	State.Mu.Lock()
	defer State.Mu.Unlock()
	if node.NodeID == "" {
		return
	}
	if ex, ok := State.ClusterNodes[node.NodeID]; !ok || ex.NodeRole == "" {
		State.ClusterNodes[node.NodeID] = node
	}
}

func markNodeHeard(id string) {
	if id == "" {
		return
	}
	State.Mu.Lock()
	defer State.Mu.Unlock()
	n := State.ClusterNodes[id]
	n.LastHeard = time.Now().UTC()
	State.ClusterNodes[id] = n
}

func isNodeActive(n NodeInfo) bool {
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
		if n.NodeRole == "IBPMonitor" && isNodeActive(n) {
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
		if n.NodeRole == "IBPDns" && isNodeActive(n) {
			cnt++
		}
	}
	return cnt
}

func StartGarbageCollection() {
	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for range t.C {
			cleanStaleNodes()
		}
	}()
}

func cleanStaleNodes() {
	now := time.Now().UTC()
	stale := 15 * time.Minute
	State.Mu.Lock()
	for id, n := range State.ClusterNodes {
		if id != State.NodeID && now.Sub(n.LastHeard) > stale {
			delete(State.ClusterNodes, id)
		}
	}
	State.Mu.Unlock()
}
