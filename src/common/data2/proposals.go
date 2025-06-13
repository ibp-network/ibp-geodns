package data2

import (
	"sync"
	"time"
)

var (
	memMu      sync.RWMutex
	memStore   = make(map[string]Proposal)
	expiryTime = 10 * time.Minute
)

func CacheProposal(p Proposal) {
	memMu.Lock()
	memStore[p.ID] = p
	memMu.Unlock()
}

func PopProposal(id string) (Proposal, bool) {
	memMu.Lock()
	defer memMu.Unlock()
	p, ok := memStore[id]
	if ok {
		delete(memStore, id)
	}
	return p, ok
}

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

func StoreProposal(p Proposal) error { CacheProposal(p); return nil }

func MarkProposalFinal(id string, yes, total int) error { _, _ = PopProposal(id); return nil }
