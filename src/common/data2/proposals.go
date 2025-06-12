package data2

import "time"

/*─────────────────────────────────────────────────────────────
  HELPER – LIVE MONITOR COUNT (LOCK HELD BY CALLER)
─────────────────────────────────────────────────────────────*/

func countActiveMonitorsLocked() int {
	n := 0
	for _, node := range State.ClusterNodes {
		if node.NodeRole == "IBPMonitor" && isNodeActive(node.NodeID) {
			n++
		}
	}
	return n
}

/*
   ──────────────────────────────────────────────────────────────────────────────
   PERSISTENCE
   ──────────────────────────────────────────────────────────────────────────────
*/

// StoreProposal upserts a proposal row so that applyOfficialChanges() can pick it
// up later without spurious “check … not found” warnings.
func StoreProposal(p Proposal) error {
	_, err := DB.Exec(`
		INSERT INTO proposals
		    (id, ipv6, domain, member, check_name, check_type, created_at)
		VALUES (?,?,?,?,?,?,?)
		ON DUPLICATE KEY UPDATE
		    ipv6       = VALUES(ipv6),
		    domain     = VALUES(domain),
		    member     = VALUES(member),
		    check_name = VALUES(check_name),
		    check_type = VALUES(check_type)
	`, p.ID, p.IsIPv6, p.Domain, p.Member, p.CheckName, p.CheckType, p.CreatedAt.UTC())
	return err
}

// MarkProposalFinal records the final vote result for auditing purposes.
func MarkProposalFinal(id string, yes, total int) error {
	_, err := DB.Exec(`
		UPDATE proposals
		   SET yes_votes   = ?,
		       total_votes = ?,
		       finalized   = 1,
		       finalized_at = NOW(6)
		 WHERE id = ?
	`, yes, total, id)
	return err
}

func isNodeActive(nodeID string) bool {
	State.Mu.RLock()
	defer State.Mu.RUnlock()

	n, ok := State.ClusterNodes[nodeID]
	if !ok {
		return false
	}
	return time.Since(n.LastHeard) < 2*time.Minute
}
