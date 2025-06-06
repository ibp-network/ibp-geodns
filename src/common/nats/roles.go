package nats

import (
	"encoding/json"
	"strings"
	"time"

	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

// EnableMonitorRole sets up the node to handle proposals and votes as a Monitor.
func EnableMonitorRole() error {
	State.SubjectPropose = "consensus.propose"
	State.SubjectVote = "consensus.vote"
	State.SubjectFinalize = "consensus.finalize"
	State.SubjectCluster = "consensus.cluster"
	State.ProposalTimeout = 12 * time.Second

	if State.Proposals == nil {
		State.Proposals = make(map[ProposalID]*ProposalTracking)
	}
	if State.ClusterNodes == nil {
		State.ClusterNodes = make(map[string]NodeInfo)
	}

	_, err := Subscribe(">", handleAllMessages)
	if err != nil {
		return err
	}

	State.ThisNode.NodeRole = "IBPMonitor"
	State.ThisNode.LastHeard = time.Now().UTC()
	State.Mu.Lock()
	State.ClusterNodes[State.NodeID] = State.ThisNode
	State.Mu.Unlock()

	StartGarbageCollection()
	StartHeartbeat() // so we broadcast ourselves regularly

	log.Log(log.Info, "[NATS] Monitor role enabled.")
	broadcastClusterJoin()
	return nil
}

// EnableDnsRole sets up the node to still listen to all messages, but not propose checks.
func EnableDnsRole() error {
	State.SubjectCluster = "consensus.cluster"

	if State.Proposals == nil {
		State.Proposals = make(map[ProposalID]*ProposalTracking)
	}
	if State.ClusterNodes == nil {
		State.ClusterNodes = make(map[string]NodeInfo)
	}

	_, err := Subscribe(">", handleAllMessages)
	if err != nil {
		return err
	}

	State.ThisNode.NodeRole = "IBPDns"
	State.ThisNode.LastHeard = time.Now().UTC()
	State.Mu.Lock()
	State.ClusterNodes[State.NodeID] = State.ThisNode
	State.Mu.Unlock()

	StartGarbageCollection()
	StartHeartbeat()

	log.Log(log.Info, "[NATS] IBPDns role enabled.")
	broadcastClusterJoin()
	return nil
}

// EnableCollatorRole sets up the node to handle usage/downtime queries.
func EnableCollatorRole() error {
	State.SubjectCluster = "consensus.cluster"

	if State.Proposals == nil {
		State.Proposals = make(map[ProposalID]*ProposalTracking)
	}
	if State.ClusterNodes == nil {
		State.ClusterNodes = make(map[string]NodeInfo)
	}

	_, err := Subscribe(">", handleAllMessages)
	if err != nil {
		return err
	}

	State.ThisNode.NodeRole = "IBPCollator"
	State.ThisNode.LastHeard = time.Now().UTC()
	State.Mu.Lock()
	State.ClusterNodes[State.NodeID] = State.ThisNode
	State.Mu.Unlock()

	StartGarbageCollection()
	StartHeartbeat()

	log.Log(log.Info, "[NATS] Collator role enabled.")
	broadcastClusterJoin()
	return nil
}

// handleAllMessages routes incoming NATS messages to the correct handler.
func handleAllMessages(m *nats.Msg) {
	subj := m.Subject

	switch {
	case subj == State.SubjectPropose:
		handleProposal(m)

	case subj == State.SubjectVote:
		handleVote(m)

	case subj == State.SubjectFinalize:
		handleFinalize(m)

	case subj == State.SubjectCluster:
		handleClusterMessage(m)

	// Monitor stats
	case subj == "monitor.stats.getDowntime":
		if State.ThisNode.NodeRole == "IBPMonitor" {
			handleMonitorStatsRequest(m)
		}
	case subj == "monitor.stats.downtimeData":
		handleMonitorStatsData(m)

	// DNS usage
	case subj == "dns.usage.getUsage":
		if State.ThisNode.NodeRole == "IBPDns" {
			handleDnsUsageRequest(m)
		}
	case subj == "dns.usage.usageData":
		handleDnsUsageData(m)

	default:
		// Possibly a reply subject, or unknown
		if strings.Contains(subj, "downtimeReply") {
			handleMonitorStatsData(m)
		} else if strings.Contains(subj, "usageReply") {
			handleDnsUsageData(m)
		} else if subj == "consensus.heartbeat" {
			handleHeartbeat(m)
		} else {
			log.Log(log.Debug, "[NATS] handleAllMessages: unhandled subject=%s", subj)
		}
	}
}

// StartGarbageCollection periodically cleans up old proposals.
func StartGarbageCollection() {
	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for range t.C {
			cleanOldProposals()
		}
	}()
}

func cleanOldProposals() {
	State.Mu.Lock()
	defer State.Mu.Unlock()

	now := time.Now().UTC()
	threshold := 900 * time.Second
	for pid, pt := range State.Proposals {
		if now.Sub(pt.Proposal.Timestamp) > threshold {
			delete(State.Proposals, pid)
			if pt.Timer != nil {
				pt.Timer.Stop()
			}
		}
	}
}

// handleClusterMessage handles join/membership messages.
func handleClusterMessage(m *nats.Msg) {
	var msg ClusterMessage
	if err := json.Unmarshal(m.Data, &msg); err != nil {
		log.Log(log.Error, "[NATS] handleClusterMessage: unmarshal error: %v", err)
		return
	}
	// Mark we heard from the sender
	markNodeHeard(msg.Sender.NodeID)

	switch msg.Type {
	case "join":
		log.Log(log.Debug, "[NATS] handleClusterMessage: got join from node=%s; adding & broadcasting membership", msg.Sender.NodeID)
		addNode(msg.Sender)
		broadcastClusterMembership()
	case "membership":
		log.Log(log.Debug, "[NATS] handleClusterMessage: got membership with %d nodes from sender=%s", len(msg.Members), msg.Sender.NodeID)
		mergeClusterMembership(msg.Members)
	default:
		log.Log(log.Warn, "[NATS] handleClusterMessage: unknown type=%s", msg.Type)
	}
}

func broadcastClusterJoin() {
	msg := ClusterMessage{
		Type:   "join",
		Sender: State.ThisNode,
	}
	data, _ := json.Marshal(msg)
	if err := Publish(State.SubjectCluster, data); err != nil {
		log.Log(log.Error, "[NATS] Failed to publish cluster join: %v", err)
	}
}

func broadcastClusterMembership() {
	State.Mu.RLock()
	var nodes []NodeInfo
	for _, node := range State.ClusterNodes {
		nodes = append(nodes, node)
	}
	State.Mu.RUnlock()

	msg := ClusterMessage{
		Type:    "membership",
		Sender:  State.ThisNode,
		Members: nodes,
	}
	data, _ := json.Marshal(msg)

	log.Log(log.Debug, "[NATS] broadcastClusterMembership: broadcasting membership of %d nodes", len(nodes))
	if err := Publish(State.SubjectCluster, data); err != nil {
		log.Log(log.Error, "[NATS] Failed to broadcast membership: %v", err)
	}
}

func mergeClusterMembership(inMembers []NodeInfo) {
	State.Mu.Lock()
	defer State.Mu.Unlock()

	countAdded := 0
	for _, m := range inMembers {
		if m.NodeID == "" {
			continue
		}
		if existing, exists := State.ClusterNodes[m.NodeID]; !exists {
			countAdded++
			log.Log(log.Debug, "[NATS] Merging node=%s role=%s into cluster", m.NodeID, m.NodeRole)
			m.LastHeard = time.Now().UTC()
			State.ClusterNodes[m.NodeID] = m
		} else {
			// update known node
			if existing.NodeID == "" {
				existing.NodeID = m.NodeID
			}
			existing.NodeRole = m.NodeRole
			if existing.LastHeard.IsZero() {
				existing.LastHeard = time.Now().UTC()
			}
			State.ClusterNodes[m.NodeID] = existing
		}
	}
	log.Log(log.Debug, "[NATS] mergeClusterMembership: added %d new node(s)", countAdded)
}

func addNode(node NodeInfo) {
	State.Mu.Lock()
	defer State.Mu.Unlock()

	if node.NodeID == "" {
		return
	}
	if _, exists := State.ClusterNodes[node.NodeID]; !exists {
		node.LastHeard = time.Now().UTC()
		State.ClusterNodes[node.NodeID] = node
		log.Log(log.Debug, "[NATS] Added node %s with role=%s to cluster", node.NodeID, node.NodeRole)
	} else {
		ex := State.ClusterNodes[node.NodeID]
		ex.NodeRole = node.NodeRole
		if ex.LastHeard.IsZero() {
			ex.LastHeard = time.Now().UTC()
		}
		State.ClusterNodes[node.NodeID] = ex
	}
}

// countNodesByRole returns how many nodes (regardless of LastHeard) match the given role.
func countNodesByRole(role string) int {
	State.Mu.RLock()
	defer State.Mu.RUnlock()

	n := 0
	for _, node := range State.ClusterNodes {
		if node.NodeRole == role {
			n++
		}
	}
	return n
}

// countActiveMonitors returns how many "IBPMonitor" nodes are considered active (LastHeard <= 2 min).
func countActiveMonitors() int {
	State.Mu.RLock()
	defer State.Mu.RUnlock()

	n := 0
	for _, node := range State.ClusterNodes {
		if node.NodeRole == "IBPMonitor" && isNodeActive(node) {
			n++
		}
	}
	return n
}

// isNodeActive returns true if the node's LastHeard is within the last 2 minutes
func isNodeActive(node NodeInfo) bool {
	if node.LastHeard.IsZero() {
		return false
	}
	if time.Since(node.LastHeard) > 2*time.Minute {
		return false
	}
	return true
}

// markNodeHeard updates lastHeard for a given nodeID if it exists
func markNodeHeard(nodeID string) {
	if nodeID == "" {
		return
	}
	State.Mu.Lock()
	defer State.Mu.Unlock()

	n, exists := State.ClusterNodes[nodeID]
	if !exists {
		// we do not have a record, create minimal
		n = NodeInfo{
			NodeID:   nodeID,
			NodeRole: "",
		}
		log.Log(log.Debug, "[NATS] markNodeHeard: discovered new nodeID=%s with no role set", nodeID)
	}
	n.LastHeard = time.Now().UTC()
	State.ClusterNodes[nodeID] = n
}

// StartHeartbeat periodically publishes a "consensus.heartbeat" from this node
func StartHeartbeat() {
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for range t.C {
			sendHeartbeat()
		}
	}()
}

func sendHeartbeat() {
	hb := NodeInfo{
		NodeID:        State.NodeID,
		PublicAddress: State.ThisNode.PublicAddress,
		ListenAddress: State.ThisNode.ListenAddress,
		ListenPort:    State.ThisNode.ListenPort,
		NodeRole:      State.ThisNode.NodeRole,
		LastHeard:     time.Now().UTC(),
	}
	data, _ := json.Marshal(hb)
	_ = Publish("consensus.heartbeat", data)
}

func handleHeartbeat(m *nats.Msg) {
	var node NodeInfo
	if err := json.Unmarshal(m.Data, &node); err != nil {
		log.Log(log.Error, "[NATS] handleHeartbeat: unmarshal error: %v", err)
		return
	}
	markNodeHeard(node.NodeID)
}
