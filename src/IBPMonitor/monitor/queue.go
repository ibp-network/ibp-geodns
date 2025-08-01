package monitor

import (
	"container/heap"
	cfg "ibp-geodns/src/common/config"
	"sync"
	"time"
)

type CheckItem struct {
	Type            string // "site", "domain", "endpoint"
	Check           cfg.Check
	Member          cfg.Member
	Service         cfg.Service
	Domain          string
	Endpoint        string
	LastExecuted    time.Time
	MinimumInterval time.Duration
	index           int // Used by heap
}

type CheckQueue struct {
	mu    sync.Mutex
	items []*CheckItem
}

func NewCheckQueue() *CheckQueue {
	cq := &CheckQueue{
		items: make([]*CheckItem, 0),
	}
	heap.Init(cq)
	return cq
}

// Priority queue implementation (heap.Interface)
func (cq *CheckQueue) Len() int { return len(cq.items) }

func (cq *CheckQueue) Less(i, j int) bool {
	// Priority based on how long overdue the check is
	now := time.Now()

	iNext := cq.items[i].LastExecuted.Add(cq.items[i].MinimumInterval)
	jNext := cq.items[j].LastExecuted.Add(cq.items[j].MinimumInterval)

	// The one that's been waiting longer gets higher priority
	return now.Sub(iNext) > now.Sub(jNext)
}

func (cq *CheckQueue) Swap(i, j int) {
	cq.items[i], cq.items[j] = cq.items[j], cq.items[i]
	cq.items[i].index = i
	cq.items[j].index = j
}

// These are required by heap.Interface - they use interface{}
func (cq *CheckQueue) Push(x interface{}) {
	item := x.(*CheckItem)
	item.index = len(cq.items)
	cq.items = append(cq.items, item)
}

func (cq *CheckQueue) Pop() interface{} {
	old := cq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	cq.items = old[0 : n-1]
	return item
}

// Thread-safe public methods with different names
func (cq *CheckQueue) Add(item *CheckItem) {
	cq.mu.Lock()
	defer cq.mu.Unlock()
	heap.Push(cq, item)
}

func (cq *CheckQueue) GetNext() *CheckItem {
	cq.mu.Lock()
	defer cq.mu.Unlock()

	if cq.Len() == 0 {
		return nil
	}

	// Find the highest priority item that's ready to run
	now := time.Now()
	for i := 0; i < cq.Len(); i++ {
		item := cq.items[i]
		nextRun := item.LastExecuted.Add(item.MinimumInterval)
		if now.After(nextRun) || now.Equal(nextRun) {
			// This check is ready to run
			heap.Remove(cq, i)
			return item
		}
	}

	// No checks are ready
	return nil
}
