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

	// Subscriptions for monitor voting
	if _, err := Subscribe(State.SubjectPropose, handleProposal); err != nil {
		return err
	}
	if _, err := Subscribe(State.SubjectVote, handleVote); err != nil {
		return err
	}
	if _, err := Subscribe(State.SubjectFinalize, handleFinalize); err != nil {
		return err
	}

	// Stats requests (Monitor <-> Collator)
	if _, err := Subscribe("monitor.stats.getDowntime", handleMonitorStatsRequest); err != nil {
		return err
	}

	// Start garbage collection for proposals
	StartGarbageCollection()
	log.Log(log.Info, "[NATS] Monitor role enabled.")
	return nil
}

// EnableDNSApiRole configures NATS subscriptions for a dnsApi node.
func EnableDNSApiRole() error {
	// Optionally subscribe to usage requests for data aggregator/collator
	if _, err := Subscribe("dns.usage.getUsage", handleDnsUsageRequest); err != nil {
		return err
	}
	log.Log(log.Info, "[NATS] DNSApi role enabled.")
	return nil
}

// EnableCollatorRole configures NATS subscriptions for a collator node.
func EnableCollatorRole() error {
	// Typically, the collator might not subscribe to anything except the responses
	// for usage data or downtime data. If we want to do pure request/reply, the collator
	// might issue requests with an Inbox. That logic can be handled externally.
	log.Log(log.Info, "[NATS] Collator role enabled.")
	return nil
}

// checkLocalStatus checks local site/domain/endpoint status.
// This is used for monitor voting.
// Return (found, status).
func checkLocalStatus(checkType string, checkName string, memberName string, domainName string, endpoint string) (bool, bool) {
	// Implementation references your local data structure or monitor's in-memory results.
	// Return (false, false) if no local result is found.
	return false, false
}

// generateProposalID creates a unique ID for proposals.
func generateProposalID() string {
	// e.g. use a UUID
	return "some-uuid" // Replace with your uuid generation logic
}

// cleanOldProposals removes old proposals from State.Proposals
func cleanOldProposals() {
	State.Mu.Lock()
	defer State.Mu.Unlock()

	now := time.Now().UTC()
	threshold := 6 * time.Second
	for pid, pt := range State.Proposals {
		if now.Sub(pt.Proposal.Timestamp) > threshold {
			delete(State.Proposals, pid)
			if pt.Timer != nil {
				pt.Timer.Stop()
			}
		}
	}
}
