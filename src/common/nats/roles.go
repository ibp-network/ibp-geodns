package nats

import (
	"encoding/json"
	"strings"
	"time"

	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

// EnableMonitorRole sets the subject constants and does Subscribe(">", handleAllMessages).
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

	// Subscribe to everything so we definitely get proposals from remote nodes
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
	StartHeartbeat()

	log.Log(log.Info, "[NATS] Monitor role enabled.")
	broadcastClusterJoin()
	return nil
}

// EnableDnsRole sets up the node as IBPDns if needed.
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

// EnableCollatorRole sets up the node as IBPCollator if needed.
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

// handleAllMessages is a universal subscription callback for ">".
func handleAllMessages(m *nats.Msg) {
	subj := m.Subject
	// Debug log for every message
	log.Log(log.Debug, "[NATS] handleAllMessages: subject=%s, dataLen=%d", subj, len(m.Data))

	switch {
	case subj == State.SubjectPropose:
		handleProposal(m)

	case subj == State.SubjectVote:
		handleVote(m)

	case subj == State.SubjectFinalize:
		handleFinalize(m)

	case subj == State.SubjectCluster:
		handleClusterMessage(m)

	case subj == "monitor.stats.getDowntime":
		// Only a monitor node responds
		if State.ThisNode.NodeRole == "IBPMonitor" {
			handleMonitorStatsRequest(m)
		}
	case subj == "monitor.stats.downtimeData":
		handleMonitorStatsData(m)

	case subj == "dns.usage.getUsage":
		// Typically only DNS roles respond, but if you want monitors to see it, keep it as is
		handleDnsUsageRequest(m)
	case subj == "dns.usage.usageData":
		handleDnsUsageData(m)

	default:
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

// handleClusterMessage handles membership messages (join/membership).
func handleClusterMessage(m *nats.Msg) {
	var msg ClusterMessage
	if err := json.Unmarshal(m.Data, &msg); err != nil {
		log.Log(log.Error, "[NATS] handleClusterMessage: unmarshal error: %v", err)
		return
	}
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
		existing, exists := State.ClusterNodes[m.NodeID]
		if !exists {
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
	existing, ok := State.ClusterNodes[node.NodeID]
	if !ok {
		node.LastHeard = time.Now().UTC()
		State.ClusterNodes[node.NodeID] = node
		log.Log(log.Debug, "[NATS] Added node %s with role=%s to cluster", node.NodeID, node.NodeRole)
	} else {
		existing.NodeRole = node.NodeRole
		if existing.LastHeard.IsZero() {
			existing.LastHeard = time.Now().UTC()
		}
		State.ClusterNodes[node.NodeID] = existing
	}
}

// Heartbeat logic:

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
