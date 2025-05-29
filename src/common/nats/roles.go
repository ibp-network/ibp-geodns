package nats

import (
	"time"

	log "ibp-geodns/src/common/logging"
)

// EnableMonitorRole configures NATS subscriptions for a serviceMonitor node.
func EnableMonitorRole() error {
	State.SubjectPropose = "consensus.propose"
	State.SubjectVote = "consensus.vote"
	State.SubjectFinalize = "consensus.finalize"
	State.SubjectCluster = "consensus.cluster"
	State.ProposalTimeout = 12 * time.Second

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

	// Start background garbage-collection for old proposals
	StartGarbageCollection()

	log.Log(log.Info, "[NATS] Monitor role enabled.")
	return nil
}

// EnableDNSApiRole configures NATS subscriptions for a dnsApi node.
func EnableDNSApiRole() error {
	// For usage requests (DNS Api <-> Collator)
	if _, err := Subscribe("dns.usage.getUsage", handleDnsUsageRequest); err != nil {
		return err
	}
	log.Log(log.Info, "[NATS] DNSApi role enabled.")
	return nil
}

// EnableCollatorRole configures NATS subscriptions for a collator node.
func EnableCollatorRole() error {
	// Typically collator doesn't subscribe to a well-known subject; it uses ephemeral inboxes.
	log.Log(log.Info, "[NATS] Collator role enabled.")
	return nil
}

// StartGarbageCollection periodically cleans old proposals
func StartGarbageCollection() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			cleanOldProposals()
		}
	}()
}

// cleanOldProposals removes proposals that exceed a time threshold
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
