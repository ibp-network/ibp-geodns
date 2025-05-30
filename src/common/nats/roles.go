package nats

import (
	"encoding/json"
	"time"

	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

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

	if _, err := Subscribe(State.SubjectPropose, handleProposal); err != nil {
		return err
	}
	if _, err := Subscribe(State.SubjectVote, handleVote); err != nil {
		return err
	}
	if _, err := Subscribe(State.SubjectFinalize, handleFinalize); err != nil {
		return err
	}
	if _, err := Subscribe(State.SubjectCluster, handleClusterMessage); err != nil {
		return err
	}
	if _, err := Subscribe("monitor.stats.getDowntime", handleMonitorStatsRequest); err != nil {
		return err
	}

	State.ThisNode.NodeRole = "IBPMonitor"
	State.ClusterNodes[State.NodeID] = State.ThisNode

	StartGarbageCollection()

	log.Log(log.Info, "[NATS] Monitor role enabled.")
	broadcastClusterJoin()
	return nil
}

func EnableDnsRole() error {
	State.SubjectCluster = "consensus.cluster"

	if State.Proposals == nil {
		State.Proposals = make(map[ProposalID]*ProposalTracking)
	}
	if State.ClusterNodes == nil {
		State.ClusterNodes = make(map[string]NodeInfo)
	}

	if _, err := Subscribe(State.SubjectCluster, handleClusterMessage); err != nil {
		return err
	}
	if _, err := Subscribe("dns.usage.getUsage", handleDnsUsageRequest); err != nil {
		return err
	}

	State.ThisNode.NodeRole = "IBPDns"
	State.ClusterNodes[State.NodeID] = State.ThisNode

	log.Log(log.Info, "[NATS] IBPDns role enabled.")
	broadcastClusterJoin()
	return nil
}

func EnableCollatorRole() error {
	State.SubjectCluster = "consensus.cluster"

	if State.Proposals == nil {
		State.Proposals = make(map[ProposalID]*ProposalTracking)
	}
	if State.ClusterNodes == nil {
		State.ClusterNodes = make(map[string]NodeInfo)
	}

	if _, err := Subscribe(State.SubjectCluster, handleClusterMessage); err != nil {
		return err
	}
	if _, err := Subscribe("dns.usage.usageData", handleDnsUsageData); err != nil {
		return err
	}
	if _, err := Subscribe("monitor.stats.downtimeData", handleMonitorStatsData); err != nil {
		return err
	}

	State.ThisNode.NodeRole = "IBPCollator"
	State.ClusterNodes[State.NodeID] = State.ThisNode

	log.Log(log.Info, "[NATS] Collator role enabled.")
	StartGarbageCollection()

	broadcastClusterJoin()
	return nil
}

func handleDnsUsageData(m *nats.Msg) {
	var resp UsageResponse
	if err := json.Unmarshal(m.Data, &resp); err != nil {
		log.Log(log.Error, "[NATS] handleDnsUsageData: unmarshal error: %v", err)
		return
	}
	log.Log(log.Debug, "[NATS] handleDnsUsageData: got %d usage records from node=%s",
		len(resp.UsageRecords), resp.NodeID)
}

func handleMonitorStatsData(m *nats.Msg) {
	var resp DowntimeResponse
	if err := json.Unmarshal(m.Data, &resp); err != nil {
		log.Log(log.Error, "[NATS] handleMonitorStatsData: unmarshal error: %v", err)
		return
	}
	log.Log(log.Debug, "[NATS] handleMonitorStatsData: got %d downtime events from node=%s",
		len(resp.Events), resp.NodeID)
}

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

func handleClusterMessage(m *nats.Msg) {
	var msg ClusterMessage
	if err := json.Unmarshal(m.Data, &msg); err != nil {
		log.Log(log.Error, "[NATS] handleClusterMessage: unmarshal error: %v", err)
		return
	}

	switch msg.Type {
	case "join":
		addNode(msg.Sender)
		broadcastClusterMembership()
	case "membership":
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

func addNode(node NodeInfo) {
	State.Mu.Lock()
	defer State.Mu.Unlock()

	if node.NodeID == "" {
		return
	}
	if _, exists := State.ClusterNodes[node.NodeID]; !exists {
		State.ClusterNodes[node.NodeID] = node
		log.Log(log.Info, "[NATS] Added node %s with role=%s to cluster", node.NodeID, node.NodeRole)
	}
}

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
