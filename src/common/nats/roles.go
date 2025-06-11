package nats

import (
	"encoding/json"
	"strings"
	"time"

	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

// We define everything about node roles, cluster membership, marking
// lastHeard, and so on.
const activeNodeWindow = 5 * time.Minute

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

	// Set our role and ensure it's properly initialized
	State.ThisNode.NodeRole = "IBPMonitor"
	State.ThisNode.NodeID = State.NodeID
	State.ThisNode.LastHeard = time.Now().UTC()

	// Add ourselves to cluster nodes BEFORE subscribing
	State.Mu.Lock()
	State.ClusterNodes[State.NodeID] = State.ThisNode
	State.Mu.Unlock()

	// Subscribe to all messages
	_, err := Subscribe(">", handleAllMessages)
	if err != nil {
		return err
	}

	StartGarbageCollection()
	startHeartbeat()

	log.Log(log.Info, "[NATS] Monitor role enabled for node=%s", State.NodeID)

	// Broadcast join multiple times to ensure it's received
	go func() {
		for i := 0; i < 3; i++ {
			broadcastClusterJoin()
			time.Sleep(500 * time.Millisecond)
		}
	}()

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

	// Set our role and ensure it's properly initialized
	State.ThisNode.NodeRole = "IBPDns"
	State.ThisNode.NodeID = State.NodeID
	State.ThisNode.LastHeard = time.Now().UTC()

	// Add ourselves to cluster nodes BEFORE subscribing
	State.Mu.Lock()
	State.ClusterNodes[State.NodeID] = State.ThisNode
	State.Mu.Unlock()

	// Subscribe to all messages
	_, err := Subscribe(">", handleAllMessages)
	if err != nil {
		return err
	}

	startHeartbeat()

	log.Log(log.Info, "[NATS] IBPDns role enabled for node=%s", State.NodeID)

	// Broadcast join multiple times to ensure it's received
	go func() {
		for i := 0; i < 3; i++ {
			broadcastClusterJoin()
			time.Sleep(500 * time.Millisecond)
		}
	}()

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

	// Set our role and ensure it's properly initialized
	State.ThisNode.NodeRole = "IBPCollator"
	State.ThisNode.NodeID = State.NodeID
	State.ThisNode.LastHeard = time.Now().UTC()

	// Add ourselves to cluster nodes BEFORE subscribing
	State.Mu.Lock()
	State.ClusterNodes[State.NodeID] = State.ThisNode
	State.Mu.Unlock()

	// Subscribe to all messages
	_, err := Subscribe(">", handleAllMessages)
	if err != nil {
		return err
	}

	StartGarbageCollection()
	startHeartbeat()

	log.Log(log.Info, "[NATS] Collator role enabled for node=%s", State.NodeID)

	// Broadcast join multiple times to ensure it's received
	go func() {
		for i := 0; i < 3; i++ {
			broadcastClusterJoin()
			time.Sleep(500 * time.Millisecond)
		}
	}()

	return nil
}

// startHeartbeat sends periodic cluster join messages to keep node visible
func startHeartbeat() {
	go func() {
		// Initial delay to let everything initialize
		time.Sleep(2 * time.Second)

		ticker := time.NewTicker(300 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			// Update our own last heard time
			State.Mu.Lock()
			if node, exists := State.ClusterNodes[State.NodeID]; exists {
				node.LastHeard = time.Now().UTC()
				State.ClusterNodes[State.NodeID] = node
			}
			State.Mu.Unlock()

			// Re-broadcast our presence
			broadcastClusterJoin()
		}
	}()
}

// handleAllMessages is the single entrypoint for the ">" subscription.
func handleAllMessages(m *nats.Msg) {
	// Process messages in a goroutine to prevent blocking the NATS handler
	go func() {
		subj := m.Subject
		dataLen := len(m.Data)

		// Always handle cluster messages regardless of role
		if subj == State.SubjectCluster {
			log.Log(log.Debug, "[NATS] Processing cluster message: subject=%s len(data)=%d", subj, dataLen)
			handleClusterMessage(m)
			return
		}

		// Role-specific message handling
		switch State.ThisNode.NodeRole {
		case "IBPMonitor":
			// Monitors handle consensus and stats requests
			switch subj {
			case State.SubjectPropose:
				log.Log(log.Debug, "[NATS] Monitor processing proposal: subject=%s len(data)=%d", subj, dataLen)
				handleProposal(m)
			case State.SubjectVote:
				log.Log(log.Debug, "[NATS] Monitor processing vote: subject=%s len(data)=%d", subj, dataLen)
				handleVote(m)
			case State.SubjectFinalize:
				log.Log(log.Debug, "[NATS] Monitor processing finalize: subject=%s len(data)=%d", subj, dataLen)
				handleFinalize(m)
			case "monitor.stats.getDowntime":
				log.Log(log.Debug, "[NATS] Monitor processing downtime request: subject=%s len(data)=%d", subj, dataLen)
				handleMonitorStatsRequest(m)
			default:
				if strings.Contains(subj, "downtimeReply") {
					handleMonitorStatsData(m)
				}
			}

		case "IBPDns":
			// DNS nodes don't participate in consensus, only handle usage requests
			switch subj {
			case "dns.usage.getUsage":
				log.Log(log.Debug, "[NATS] DNS processing usage request: subject=%s len(data)=%d", subj, dataLen)
				handleDnsUsageRequest(m)
			default:
				if strings.Contains(subj, "usageReply") {
					handleDnsUsageData(m)
				}
			}

		case "IBPCollator":
			// Collators only handle responses to their requests
			switch {
			case subj == "monitor.stats.downtimeData" || strings.Contains(subj, "downtimeReply"):
				log.Log(log.Debug, "[NATS] Collator processing downtime data: subject=%s len(data)=%d", subj, dataLen)
				handleMonitorStatsData(m)
			case subj == "dns.usage.usageData" || strings.Contains(subj, "usageReply"):
				log.Log(log.Debug, "[NATS] Collator processing usage data: subject=%s len(data)=%d", subj, dataLen)
				handleDnsUsageData(m)
			}
		}
	}()
}

// broadcastClusterJoin publishes "join"
func broadcastClusterJoin() {
	// Ensure ThisNode has all required fields
	if State.ThisNode.NodeID == "" {
		log.Log(log.Error, "[NATS] broadcastClusterJoin: ThisNode.NodeID is empty!")
		return
	}
	if State.ThisNode.NodeRole == "" {
		log.Log(log.Error, "[NATS] broadcastClusterJoin: ThisNode.NodeRole is empty!")
		return
	}

	msg := ClusterMessage{
		Type:   "join",
		Sender: State.ThisNode,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Log(log.Error, "[NATS] Failed to marshal cluster join message: %v", err)
		return
	}

	log.Log(log.Debug, "[NATS] Broadcasting cluster join for node=%s role=%s to subject=%s",
		State.ThisNode.NodeID, State.ThisNode.NodeRole, State.SubjectCluster)

	if err := Publish(State.SubjectCluster, data); err != nil {
		log.Log(log.Error, "[NATS] Failed to publish cluster join: %v", err)
	} else {
		log.Log(log.Debug, "[NATS] Successfully published cluster join message")
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

	// Check if sender has required fields
	if msg.Sender.NodeID == "" {
		log.Log(log.Error, "[NATS] handleClusterMessage: received message with empty NodeID")
		return
	}

	// Mark we heard from the cluster sender
	markNodeHeard(msg.Sender.NodeID)

	switch msg.Type {
	case "join":
		//log.Log(log.Debug, "[NATS] handleClusterMessage: got join from node=%s role=%s", msg.Sender.NodeID, msg.Sender.NodeRole)
		addNode(msg.Sender)
		// Always broadcast membership when we get a join
		broadcastClusterMembership()

	case "membership":
		//log.Log(log.Debug, "[NATS] handleClusterMessage: got membership with %d nodes from sender=%s role=%s", len(msg.Members), msg.Sender.NodeID, msg.Sender.NodeRole)
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
	countUpdated := 0
	for _, m := range inMembers {
		if m.NodeID == "" {
			continue
		}
		existing, exists := State.ClusterNodes[m.NodeID]
		if !exists {
			State.ClusterNodes[m.NodeID] = m
			countAdded++
			log.Log(log.Debug, "[NATS] Added new node=%s role=%s to cluster", m.NodeID, m.NodeRole)
		} else if existing.NodeRole == "" && m.NodeRole != "" {
			// Update node if we didn't have its role before
			State.ClusterNodes[m.NodeID] = m
			countUpdated++
			log.Log(log.Debug, "[NATS] Updated node=%s with role=%s", m.NodeID, m.NodeRole)
		}
	}
}

// addNode adds a single node
func addNode(node NodeInfo) {
	State.Mu.Lock()
	defer State.Mu.Unlock()

	if node.NodeID == "" {
		return
	}

	existing, exists := State.ClusterNodes[node.NodeID]
	if !exists {
		State.ClusterNodes[node.NodeID] = node
		log.Log(log.Debug, "[NATS] Added node=%s role=%s to cluster", node.NodeID, node.NodeRole)
	} else if existing.NodeRole == "" && node.NodeRole != "" {
		// Update if we have better info
		State.ClusterNodes[node.NodeID] = node
		log.Log(log.Debug, "[NATS] Updated node=%s with role=%s", node.NodeID, node.NodeRole)
	}
}

// markNodeHeard updates clusterNodes[nodeID].LastHeard to now.
// If nodeID is missing from ClusterNodes, we add it with blank role.
func markNodeHeard(nodeID string) {
	if nodeID == "" {
		return
	}
	State.Mu.Lock()
	defer State.Mu.Unlock()

	ni, ok := State.ClusterNodes[nodeID]
	if !ok {
		log.Log(log.Warn, "[NATS] markNodeHeard: discovered new nodeID=%s with no role set - node should announce itself!", nodeID)
		ni = NodeInfo{
			NodeID:        nodeID,
			NodeRole:      "", // Empty role because node hasn't announced itself
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
	threshold := 10 * time.Minute
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
	staleAfter := 15 * time.Minute

	State.Mu.Lock()
	defer State.Mu.Unlock()

	var toRemove []string
	for nodeID, node := range State.ClusterNodes {
		if nodeID == State.NodeID {
			continue
		}
		if !node.LastHeard.IsZero() && now.Sub(node.LastHeard) > staleAfter {
			toRemove = append(toRemove, nodeID)
		}
	}

	// Remove stale nodes
	for _, nodeID := range toRemove {
		node := State.ClusterNodes[nodeID]
		log.Log(log.Debug, "[NATS] Removing stale node=%s role=%s lastHeard=%v (age=%v)",
			nodeID, node.NodeRole, node.LastHeard, now.Sub(node.LastHeard))
		delete(State.ClusterNodes, nodeID)
	}

	if len(toRemove) > 0 {
		// Log current active nodes
		activeCount := 0
		for nodeID, node := range State.ClusterNodes {
			if isNodeActive(node) {
				activeCount++
				log.Log(log.Debug, "[NATS] Active node: %s role=%s", nodeID, node.NodeRole)
			}
		}
		log.Log(log.Debug, "[NATS] After cleanup: %d active nodes, %d total nodes", activeCount, len(State.ClusterNodes))
	}
}

// CountActiveMonitors returns how many IBPMonitor nodes are not stale.
func CountActiveMonitors() int {
	State.Mu.RLock()
	defer State.Mu.RUnlock()

	n := 0
	for _, node := range State.ClusterNodes {
		if node.NodeRole == "IBPMonitor" && IsNodeActive(node) {
			n++
		}
	}
	log.Log(log.Debug, "[NATS] CountActiveMonitors: found %d active monitors", n)
	return n
}

// IsNodeActive checks LastHeard with 2-min threshold
func IsNodeActive(ni NodeInfo) bool {
	if ni.NodeID == "" {
		return false
	}
	if ni.LastHeard.IsZero() {
		return false
	}
	return time.Since(ni.LastHeard) < activeNodeWindow
}

// CountActiveDns returns how many IBPDns nodes are active
func CountActiveDns() int {
	State.Mu.RLock()
	defer State.Mu.RUnlock()

	n := 0
	for _, node := range State.ClusterNodes {
		if node.NodeRole == "IBPDns" && IsNodeActive(node) {
			n++
		}
	}
	log.Log(log.Debug, "[NATS] CountActiveDns: found %d active DNS nodes", n)
	return n
}

// Expose internal functions for package use
var (
	countActiveMonitors = CountActiveMonitors
	countActiveDns      = CountActiveDns
	isNodeActive        = IsNodeActive
)
