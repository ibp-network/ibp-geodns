package nats

import (
	"encoding/json"
	"strings"
	"time"

	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

// EnableMonitorRole sets up the current node as an IBPMonitor
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
	// Ensure subscription is active before proceeding, so we don't miss any initial messages.
	if flushErr := Flush(); flushErr != nil {
		return flushErr
	}

	State.ThisNode.NodeRole = "IBPMonitor"
	State.ClusterNodes[State.NodeID] = State.ThisNode
	StartGarbageCollection()
	log.Log(log.Info, "[NATS] Monitor role enabled.")

	broadcastClusterJoin()
	return nil
}

// EnableDnsRole sets up the current node as an IBPDns
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

	_, err := Subscribe(">", handleAllMessages)
	if err != nil {
		return err
	}
	// Flush subscription to avoid losing messages
	if flushErr := Flush(); flushErr != nil {
		return flushErr
	}

	State.ThisNode.NodeRole = "IBPDns"
	State.ClusterNodes[State.NodeID] = State.ThisNode

	log.Log(log.Info, "[NATS] IBPDns role enabled.")
	broadcastClusterJoin()
	return nil
}

// EnableCollatorRole sets up the current node as an IBPCollator
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

	_, err := Subscribe(">", handleAllMessages)
	if err != nil {
		return err
	}
	// Flush subscription to avoid losing messages
	if flushErr := Flush(); flushErr != nil {
		return flushErr
	}

	State.ThisNode.NodeRole = "IBPCollator"
	State.ClusterNodes[State.NodeID] = State.ThisNode
	StartGarbageCollection()
	log.Log(log.Info, "[NATS] Collator role enabled.")

	broadcastClusterJoin()
	return nil
}

// handleAllMessages is the single entrypoint for the ">" subscription.
// We parse the subject and choose which messages to handle based on role.
func handleAllMessages(m *nats.Msg) {
	subj := m.Subject
	dataLen := len(m.Data)

	log.Log(log.Debug, "[NATS] Subscription received subject=%s len(data)=%d", subj, dataLen)

	// 1) Cluster membership messages (all node roles handle these)
	if subj == State.SubjectCluster {
		handleClusterMessage(m)
		return
	}

	// 2) consensus.* messages => only IBPMonitor handles them
	if subj == State.SubjectPropose {
		if State.ThisNode.NodeRole == "IBPMonitor" {
			handleProposal(m)
		}
		return
	}
	if subj == State.SubjectVote {
		if State.ThisNode.NodeRole == "IBPMonitor" {
			handleVote(m)
		}
		return
	}
	if subj == State.SubjectFinalize {
		if State.ThisNode.NodeRole == "IBPMonitor" {
			handleFinalize(m)
		}
		return
	}

	// 3) Monitor stats => only IBPMonitor responds to 'getDowntime', but all can receive 'downtimeData'
	if subj == "monitor.stats.getDowntime" {
		if State.ThisNode.NodeRole == "IBPMonitor" {
			handleMonitorStatsRequest(m)
		}
		return
	}
	if subj == "monitor.stats.downtimeData" {
		// Collator or Monitor can handle it, so no role check
		handleMonitorStatsData(m)
		return
	}

	// 4) DNS usage => only IBPDns responds to 'getUsage', but all can receive 'usageData'
	if subj == "dns.usage.getUsage" {
		if State.ThisNode.NodeRole == "IBPDns" {
			handleDnsUsageRequest(m)
		}
		return
	}
	if subj == "dns.usage.usageData" {
		handleDnsUsageData(m)
		return
	}

	// 5) Possibly request/reply for "downtimeReply" or "usageReply"
	if strings.Contains(subj, "downtimeReply") {
		handleMonitorStatsData(m)
		return
	}
	if strings.Contains(subj, "usageReply") {
		handleDnsUsageData(m)
		return
	}

	log.Log(log.Debug, "[NATS] handleAllMessages: unhandled subject=%s", subj)
}

// broadcastClusterJoin publishes "join"
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

// broadcastClusterMembership publishes "membership"
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

// handleClusterMessage processes "join" or "membership"
func handleClusterMessage(m *nats.Msg) {
	var msg ClusterMessage
	if err := json.Unmarshal(m.Data, &msg); err != nil {
		log.Log(log.Error, "[NATS] handleClusterMessage: unmarshal error: %v", err)
		return
	}

	// Mark we heard from the cluster sender
	markNodeHeard(msg.Sender.NodeID)

	switch msg.Type {
	case "join":
		log.Log(log.Debug, "[NATS] handleClusterMessage: got join from node=%s; adding & broadcasting membership", msg.Sender.NodeID)
		addNode(msg.Sender)
		broadcastClusterMembership()

	case "membership":
		log.Log(log.Debug, "[NATS] handleClusterMessage: got membership with %d nodes from sender=%s",
			len(msg.Members), msg.Sender.NodeID)
		mergeClusterMembership(msg.Members)

	default:
		log.Log(log.Warn, "[NATS] handleClusterMessage: unknown type=%s", msg.Type)
	}
}

// mergeClusterMembership merges an incoming membership list
func mergeClusterMembership(inMembers []NodeInfo) {
	State.Mu.Lock()
	defer State.Mu.Unlock()

	countAdded := 0
	for _, m := range inMembers {
		if m.NodeID == "" {
			continue
		}
		if _, exists := State.ClusterNodes[m.NodeID]; !exists {
			State.ClusterNodes[m.NodeID] = m
			countAdded++
			log.Log(log.Debug, "[NATS] Merging node=%s role=%s into cluster", m.NodeID, m.NodeRole)
		}
	}
	log.Log(log.Debug, "[NATS] mergeClusterMembership: added %d new node(s)", countAdded)

	// ADDITIONAL DEBUG: Log entire membership
	log.Log(log.Debug, "[NATS] Current membership count is %d", len(State.ClusterNodes))
	for idKey, nodeVal := range State.ClusterNodes {
		log.Log(log.Debug,
			"[NATS]   -> NodeID=%s Role=%s LastHeard=%v",
			idKey, nodeVal.NodeRole, nodeVal.LastHeard)
	}
}

// addNode adds a single node
func addNode(node NodeInfo) {
	State.Mu.Lock()
	defer State.Mu.Unlock()

	if node.NodeID == "" {
		return
	}
	if _, exists := State.ClusterNodes[node.NodeID]; !exists {
		State.ClusterNodes[node.NodeID] = node
		log.Log(log.Debug, "[NATS] Added node=%s role=%s to cluster", node.NodeID, node.NodeRole)
	}
}

// markNodeHeard updates clusterNodes[nodeID].LastHeard to now.
func markNodeHeard(nodeID string) {
	if nodeID == "" {
		return
	}
	State.Mu.Lock()
	defer State.Mu.Unlock()

	ni, ok := State.ClusterNodes[nodeID]
	if !ok {
		log.Log(log.Debug, "[NATS] markNodeHeard: discovered new nodeID=%s with no role set", nodeID)
		ni = NodeInfo{
			NodeID:        nodeID,
			NodeRole:      "",
			ListenAddress: "",
			ListenPort:    "",
		}
	}
	ni.LastHeard = time.Now().UTC()
	State.ClusterNodes[nodeID] = ni
}

// StartGarbageCollection periodically removes old proposals or stale nodes
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
	threshold := 15 * time.Minute
	for pid, pt := range State.Proposals {
		if now.Sub(pt.Proposal.Timestamp) > threshold {
			delete(State.Proposals, pid)
			if pt.Timer != nil {
				pt.Timer.Stop()
			}
		}
	}
}

func cleanStaleNodes() {
	// remove nodes not heard from in e.g. 2 minutes
	now := time.Now().UTC()
	staleAfter := 2 * time.Minute

	State.Mu.Lock()
	defer State.Mu.Unlock()

	for nodeID, node := range State.ClusterNodes {
		if nodeID == State.NodeID {
			continue
		}
		if !node.LastHeard.IsZero() && now.Sub(node.LastHeard) > staleAfter {
			log.Log(log.Debug, "[NATS] cleanStaleNodes: removing stale node=%s role=%s lastHeard=%v",
				nodeID, node.NodeRole, node.LastHeard)
			delete(State.ClusterNodes, nodeID)
		}
	}
}

// countActiveMonitors returns how many IBPMonitor nodes are not stale.
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

// isNodeActive checks LastHeard with 2-min threshold
func isNodeActive(ni NodeInfo) bool {
	if ni.NodeID == "" {
		return false
	}
	if ni.LastHeard.IsZero() {
		return false
	}
	if time.Since(ni.LastHeard) > 2*time.Minute {
		return false
	}
	return true
}

// countActiveDns is optional if you want to do a similar majority-based finalization for DNS
func countActiveDns() int {
	State.Mu.RLock()
	defer State.Mu.RUnlock()

	n := 0
	for _, node := range State.ClusterNodes {
		if node.NodeRole == "IBPDns" && isNodeActive(node) {
			n++
		}
	}
	return n
}
