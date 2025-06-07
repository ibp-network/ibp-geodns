package nats

import (
	log "ibp-geodns/src/common/logging"
	"time"
)

// markNodeHeard updates LastHeard on the given nodeID to now.
// If the node doesn't exist, we skip or create a minimal entry.
func markNodeHeard(nodeID string) {
	if nodeID == "" {
		return
	}
	State.Mu.Lock()
	defer State.Mu.Unlock()
	node, exists := State.ClusterNodes[nodeID]
	if !exists {
		// If we do not have that node, create it with minimal info
		node = NodeInfo{
			NodeID:    nodeID,
			NodeRole:  "",
			LastHeard: time.Now().UTC(),
		}
		State.ClusterNodes[nodeID] = node
		log.Log(log.Debug, "[NATS] markNodeHeard: discovered new nodeID=%s with no role set", nodeID)
	} else {
		// Just update lastHeard
		node.LastHeard = time.Now().UTC()
		State.ClusterNodes[nodeID] = node
	}
}

// isNodeActive returns true if node.LastHeard was within, e.g., 5 minutes
func isNodeActive(node NodeInfo) bool {
	// Let's define a 5-minute threshold for “active”
	// Adjust as needed
	cutoff := time.Now().UTC().Add(-5 * time.Minute)
	return node.LastHeard.After(cutoff)
}

// countActiveMonitors returns how many cluster nodes are monitors and active
func countActiveMonitors() int {
	State.Mu.RLock()
	defer State.Mu.RUnlock()

	activeCount := 0
	for _, nd := range State.ClusterNodes {
		if nd.NodeRole == "IBPMonitor" && isNodeActive(nd) {
			activeCount++
		}
	}
	return activeCount
}
