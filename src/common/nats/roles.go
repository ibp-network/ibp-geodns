package nats

import (
	"encoding/json"
	"time"

	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

// EnableMonitorRole configures NATS subscriptions for a serviceMonitor node.
// This includes:
//   - Subscribing to consensus proposals/votes/finalize
//   - Subscribing to cluster messages
//   - Subscribing to downtime requests (monitor.stats.getDowntime)
//   - Broadcasting its own join event so other nodes see it
//   - Starting garbage collection of stale proposals
func EnableMonitorRole() error {
	// Set up standard consensus subjects
	State.SubjectPropose = "consensus.propose"
	State.SubjectVote = "consensus.vote"
	State.SubjectFinalize = "consensus.finalize"
	State.SubjectCluster = "consensus.cluster"
	State.ProposalTimeout = 12 * time.Second

	// Ensure proposals/cluster maps exist
	if State.Proposals == nil {
		State.Proposals = make(map[ProposalID]*ProposalTracking)
	}
	if State.ClusterNodes == nil {
		State.ClusterNodes = make(map[string]NodeInfo)
	}

	// Subscribe to proposals, votes, finalization
	if _, err := Subscribe(State.SubjectPropose, handleProposal); err != nil {
		return err
	}
	if _, err := Subscribe(State.SubjectVote, handleVote); err != nil {
		return err
	}
	if _, err := Subscribe(State.SubjectFinalize, handleFinalize); err != nil {
		return err
	}

	// Subscribe to cluster membership
	if _, err := Subscribe(State.SubjectCluster, handleClusterMessage); err != nil {
		return err
	}

	// Subscribe to requests for downtime (monitors respond to "monitor.stats.getDowntime")
	if _, err := Subscribe("monitor.stats.getDowntime", handleMonitorStatsRequest); err != nil {
		return err
	}

	// Set node role and store in cluster
	State.ThisNode.NodeRole = "IBPMonitor"
	State.ClusterNodes[State.NodeID] = State.ThisNode

	// Begin proposal garbage collection
	StartGarbageCollection()

	log.Log(log.Info, "[NATS] Monitor role enabled.")

	// Announce our presence to the cluster
	broadcastClusterJoin()
	return nil
}

// EnableDnsRole configures NATS subscriptions for a DNS node (IBPDns).
// This includes:
//   - Subscribing to "consensus.cluster" to learn about membership
//   - Subscribing to "dns.usage.getUsage" so it can handle usage requests
//   - Broadcasting its join event
func EnableDnsRole() error {
	State.SubjectCluster = "consensus.cluster"

	if State.Proposals == nil {
		State.Proposals = make(map[ProposalID]*ProposalTracking)
	}
	if State.ClusterNodes == nil {
		State.ClusterNodes = make(map[string]NodeInfo)
	}

	// Subscribe to cluster membership
	if _, err := Subscribe(State.SubjectCluster, handleClusterMessage); err != nil {
		return err
	}

	// DNS node should handle usage requests
	if _, err := Subscribe("dns.usage.getUsage", handleDnsUsageRequest); err != nil {
		return err
	}

	State.ThisNode.NodeRole = "IBPDns"
	State.ClusterNodes[State.NodeID] = State.ThisNode

	log.Log(log.Info, "[NATS] IBPDns role enabled.")

	// Broadcast our join
	broadcastClusterJoin()
	return nil
}

// EnableCollatorRole configures NATS subscriptions for a Collator node.
// This includes:
//   - Subscribing to "consensus.cluster" to learn about membership
//   - Subscribing to fallback usage data ("dns.usage.usageData")
//   - Subscribing to fallback downtime data ("monitor.stats.downtimeData")
//   - Broadcasting its join event so older nodes will see it
//   - (Optionally) garbage collection if you want to track proposals in collator
func EnableCollatorRole() error {
	State.SubjectCluster = "consensus.cluster"

	if State.Proposals == nil {
		State.Proposals = make(map[ProposalID]*ProposalTracking)
	}
	if State.ClusterNodes == nil {
		State.ClusterNodes = make(map[string]NodeInfo)
	}

	// Subscribe to cluster membership
	if _, err := Subscribe(State.SubjectCluster, handleClusterMessage); err != nil {
		return err
	}

	// Collator might receive fallback usage or downtime data if no ephemeral reply is used
	if _, err := Subscribe("dns.usage.usageData", handleDnsUsageData); err != nil {
		return err
	}
	if _, err := Subscribe("monitor.stats.downtimeData", handleMonitorStatsData); err != nil {
		return err
	}

	State.ThisNode.NodeRole = "IBPCollator"
	State.ClusterNodes[State.NodeID] = State.ThisNode

	log.Log(log.Info, "[NATS] Collator role enabled.")

	// (Optional) start garbage collection if collator also tracks proposals
	StartGarbageCollection()

	// Announce our presence
	broadcastClusterJoin()
	return nil
}

// handleDnsUsageData is a fallback for usage data broadcast (if no reply subject).
// Collators might want to gather usage data from DNS nodes in broadcast form.
func handleDnsUsageData(m *nats.Msg) {
	var resp UsageResponse
	if err := json.Unmarshal(m.Data, &resp); err != nil {
		log.Log(log.Error, "[NATS] handleDnsUsageData: unmarshal error: %v", err)
		return
	}
	log.Log(log.Debug, "[NATS] handleDnsUsageData: got %d usage records from node=%s",
		len(resp.UsageRecords), resp.NodeID)

	// TODO: Merge or store usage data in collator aggregator
}

// handleMonitorStatsData is a fallback for downtime data broadcast
func handleMonitorStatsData(m *nats.Msg) {
	var resp DowntimeResponse
	if err := json.Unmarshal(m.Data, &resp); err != nil {
		log.Log(log.Error, "[NATS] handleMonitorStatsData: unmarshal error: %v", err)
		return
	}
	log.Log(log.Debug, "[NATS] handleMonitorStatsData: got %d downtime events from node=%s",
		len(resp.Events), resp.NodeID)

	// TODO: Merge or store downtime events in collator aggregator
}

// StartGarbageCollection periodically cleans up old proposals
func StartGarbageCollection() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			cleanOldProposals()
		}
	}()
}

// cleanOldProposals removes proposals older than 900 seconds
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

// handleClusterMessage processes membership messages on "consensus.cluster".
func handleClusterMessage(m *nats.Msg) {
	var msg ClusterMessage
	if err := json.Unmarshal(m.Data, &msg); err != nil {
		log.Log(log.Error, "[NATS] handleClusterMessage: unmarshal error: %v", err)
		return
	}

	switch msg.Type {
	case "join":
		// Another node joined
		addNode(msg.Sender)
		broadcastClusterMembership()

	case "membership":
		// Another node is broadcasting membership; merge it
		mergeClusterMembership(msg.Members)

	default:
		log.Log(log.Warn, "[NATS] handleClusterMessage: unknown type=%s", msg.Type)
	}
}

// broadcastClusterJoin announces THIS node joined
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

// broadcastClusterMembership sends out a membership list so joiner can merge it
func broadcastClusterMembership() {
	State.Mu.RLock()
	defer State.Mu.RUnlock()

	var nodes []NodeInfo
	for _, node := range State.ClusterNodes {
		nodes = append(nodes, node)
	}

	msg := ClusterMessage{
		Type:    "membership",
		Sender:  State.ThisNode,
		Members: nodes,
	}
	data, _ := json.Marshal(msg)
	if err := Publish(State.SubjectCluster, data); err != nil {
		log.Log(log.Error, "[NATS] Failed to broadcast membership: %v", err)
	}
}

// mergeClusterMembership merges an inbound membership list into our cluster map
func mergeClusterMembership(inMembers []NodeInfo) {
	State.Mu.Lock()
	defer State.Mu.Unlock()

	for _, m := range inMembers {
		if m.NodeID == "" {
			continue
		}
		State.ClusterNodes[m.NodeID] = m
	}
}

// addNode inserts a single node if not present
func addNode(node NodeInfo) {
	State.Mu.Lock()
	defer State.Mu.Unlock()

	if node.NodeID == "" {
		return
	}
	if _, exists := State.ClusterNodes[node.NodeID]; !exists {
		State.ClusterNodes[node.NodeID] = node
		log.Log(log.Info,
			"[NATS] Added node %s with role=%s to cluster",
			node.NodeID, node.NodeRole)
	}
}

// countNodesByRole returns how many nodes currently have the given role
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
