package data2

import (
	"sync"
	"time"
)

/* ------------------------------------------------------------------------- */
/*  IN‑MEMORY PROPOSAL CACHE – NO DATABASE DEPENDENCY                         */
/* ------------------------------------------------------------------------- */

var (
	memMu      sync.RWMutex
	memStore   = make(map[string]Proposal)
	expiryTime = 10 * time.Minute
)

// CacheProposal keeps a proposal until it is finalised or times out.
func CacheProposal(p Proposal) { // <- used by cmd_Consensus
	memMu.Lock()
	memStore[p.ID] = p
	memMu.Unlock()
}

// PopProposal fetches & removes an entry once it is finalised.
func PopProposal(id string) (Proposal, bool) { // <- used by cmd_Consensus
	memMu.Lock()
	defer memMu.Unlock()
	p, ok := memStore[id]
	if ok {
		delete(memStore, id)
	}
	return p, ok
}

// ExpireStaleProposals is called by a janitor goroutine in nats/collator_services.go.
func ExpireStaleProposals() {
	cut := time.Now().UTC().Add(-expiryTime)
	memMu.Lock()
	for id, p := range memStore {
		if p.CreatedAt.Before(cut) {
			delete(memStore, id)
		}
	}
	memMu.Unlock()
}

/* ------------------------------------------------------------------------- */
/*  BACKWARD‑COMPATIBILITY SHIMS ( NO‑OP FOR MYSQL )                          */
/* ------------------------------------------------------------------------- */

// Legacy functions still referenced in cmd_Consensus.go; they now just
// forward to the RAM cache so existing code compiles unchanged.

func StoreProposal(p Proposal) error { CacheProposal(p); return nil }

func MarkProposalFinal(id string, yes, total int) error { _, _ = PopProposal(id); return nil }
