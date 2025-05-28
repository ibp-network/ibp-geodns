package nats

import (
	"time"

	log "ibp-geodns/src/common/logging"
)

// EnableMonitorRole configures NATS subscriptions for a serviceMonitor node.
func EnableMonitorRole() error {
	// This ensures the node is recognized in the cluster
	State.SubjectPropose = "consensus.propose"
	State.SubjectVote = "consensus.vote"
	State.SubjectFinalize = "consensus.finalize"
	State.SubjectCluster = "consensus.cluster"
	State.ProposalTimeout = 12 * time.Second // or from config

	State.Proposals = make(map[ProposalID]*ProposalTracking)
	State.ClusterNodes = make(map[string]NodeInfo)
	State.ClusterNodes[State.NodeID] = State.ThisNode

	// Subscriptions for monitor voting (from monitorVoting.go)
	if _, err := Subscribe(State.SubjectPropose, handleProposal); err != nil {
		return err
	}
	if _, err := Subscribe(State.SubjectVote, handleVote); err != nil {
		return err
	}
	if _, err := Subscribe(State.SubjectFinalize, handleFinalize); err != nil {
		return err
	}

	// Stats requests (Monitor <-> Collator) from monitorStats.go
	if _, err := Subscribe("monitor.stats.getDowntime", handleMonitorStatsRequest); err != nil {
		return err
	}

	// Start background garbage-collection for old proposals
	StartGarbageCollection()

	log.Log(log.Info, "[NATS] Monitor role enabled.")
	return nil
}

// EnableDNSApiRole configures NATS subscriptions for a dnsApi node.
func EnableDNSApiRole() error {
	// Subscriptions for usage requests (DNS Api <-> Collator) from dnsUsage.go
	if _, err := Subscribe("dns.usage.getUsage", handleDnsUsageRequest); err != nil {
		return err
	}
	log.Log(log.Info, "[NATS] DNSApi role enabled.")
	return nil
}

// EnableCollatorRole configures NATS subscriptions for a collator node.
func EnableCollatorRole() error {
	// Typically, the collator might not subscribe to anything except ephemeral inboxes
	// for usage or downtime responses. So we might not subscribe to a well-known subject.
	log.Log(log.Info, "[NATS] Collator role enabled.")
	return nil
}

// StartGarbageCollection runs a periodic cleanup of old proposals.
// Called automatically in EnableMonitorRole().
func StartGarbageCollection() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			cleanOldProposals()
		}
	}()
}

// cleanOldProposals removes proposals from State.Proposals that exceed a time threshold.
func cleanOldProposals() {
	State.Mu.Lock()
	defer State.Mu.Unlock()

	now := time.Now().UTC()
	threshold := 6 * time.Second // or from config
	for pid, pt := range State.Proposals {
		if now.Sub(pt.Proposal.Timestamp) > threshold {
			delete(State.Proposals, pid)
			if pt.Timer != nil {
				pt.Timer.Stop()
			}
		}
	}
}

// checkLocalStatus checks local site/domain/endpoint status for monitorVoting.go usage.
// Return (found, status).
func checkLocalStatus(checkType string, checkName string, memberName string, domainName string, endpoint string) (bool, bool) {
	// Implementation references your local data structure or monitor's in-memory results
	// Return (false, false) if no local result is found
	return false, false
}

// generateProposalID creates a unique ID for proposals, e.g. using a UUID or random string.
func generateProposalID() string {
	// This can be replaced with a real UUID generator if needed.
	return "some-random-id"
}
