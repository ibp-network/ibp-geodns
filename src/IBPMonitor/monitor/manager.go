package monitor

import (
	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	"sync"
	"time"
)

type CheckManager struct {
	workers      []*Worker
	checkQueue   *CheckQueue
	numWorkers   int
	separationMs int
	shutdownCh   chan struct{}
	wg           sync.WaitGroup
}

type Worker struct {
	id         int
	manager    *CheckManager
	startDelay time.Duration
	ticker     *time.Ticker
}

func NewCheckManager() *CheckManager {
	c := cfg.GetConfig()
	numWorkers := c.Local.CheckWorkers.NumWorkers
	if numWorkers <= 0 {
		numWorkers = 10 // default
	}

	separationMs := c.Local.CheckWorkers.SeparationInterval
	if separationMs <= 0 {
		separationMs = 1000 // default 1 second
	}

	return &CheckManager{
		workers:      make([]*Worker, numWorkers),
		checkQueue:   NewCheckQueue(),
		numWorkers:   numWorkers,
		separationMs: separationMs,
		shutdownCh:   make(chan struct{}),
	}
}

func (cm *CheckManager) Start() {
	log.Log(log.Info, "Starting CheckManager with %d workers, %dms separation",
		cm.numWorkers, cm.separationMs)

	// Initialize all checks in the queue
	cm.initializeChecks()

	// Start workers with staggered delays
	for i := 0; i < cm.numWorkers; i++ {
		worker := &Worker{
			id:         i,
			manager:    cm,
			startDelay: time.Duration(i*cm.separationMs) * time.Millisecond,
		}
		cm.workers[i] = worker
		cm.wg.Add(1)
		go worker.run()
	}

	// Start the queue maintenance routine
	cm.wg.Add(1)
	go cm.maintainQueue()
}

func (cm *CheckManager) Stop() {
	log.Log(log.Info, "Stopping CheckManager...")
	close(cm.shutdownCh)
	cm.wg.Wait()
	log.Log(log.Info, "CheckManager stopped")
}

func (cm *CheckManager) initializeChecks() {
	c := cfg.GetConfig()

	for _, check := range c.Local.Checks {
		if check.Enabled != 1 {
			continue
		}

		switch check.CheckType {
		case "site":
			cm.initializeSiteChecks(check)
		case "domain":
			cm.initializeDomainChecks(check)
		case "endpoint":
			cm.initializeEndpointChecks(check)
		}
	}

	log.Log(log.Info, "Initialized %d checks in queue", cm.checkQueue.Len())
}

func (cm *CheckManager) initializeSiteChecks(check cfg.Check) {
	c := cfg.GetConfig()
	for _, member := range c.Members {
		if member.Service.Active == 1 && !member.Override {
			item := &CheckItem{
				Type:            "site",
				Check:           check,
				Member:          member,
				LastExecuted:    time.Time{}, // Never executed
				MinimumInterval: time.Duration(check.MinimumInterval) * time.Second,
			}
			cm.checkQueue.Add(item)
		}
	}
}

func (cm *CheckManager) initializeDomainChecks(check cfg.Check) {
	c := cfg.GetConfig()
	for svcName, svc := range c.Services {
		if !isCheckValidForServiceType(check.Name, "domain", svc.Configuration.ServiceType) {
			continue
		}

		for _, mem := range c.Members {
			if mem.Service.Active == 1 && !mem.Override &&
				mem.Membership.Level >= svc.Configuration.LevelRequired &&
				assignedToService(svcName, mem) {

				domains := extractDomains(svc)
				for domain := range domains {
					item := &CheckItem{
						Type:            "domain",
						Check:           check,
						Member:          mem,
						Service:         svc,
						Domain:          domain,
						LastExecuted:    time.Time{},
						MinimumInterval: time.Duration(check.MinimumInterval) * time.Second,
					}
					cm.checkQueue.Add(item)
				}
			}
		}
	}
}

func (cm *CheckManager) initializeEndpointChecks(check cfg.Check) {
	c := cfg.GetConfig()
	for svcName, svc := range c.Services {
		if !isCheckValidForServiceType(check.Name, "endpoint", svc.Configuration.ServiceType) {
			continue
		}

		for _, mem := range c.Members {
			if mem.Service.Active == 1 && !mem.Override &&
				mem.Membership.Level >= svc.Configuration.LevelRequired &&
				assignedToService(svcName, mem) {

				for _, prov := range svc.Providers {
					for _, endpoint := range prov.RpcUrls {
						item := &CheckItem{
							Type:            "endpoint",
							Check:           check,
							Member:          mem,
							Service:         svc,
							Endpoint:        endpoint,
							Domain:          parseUrlForDomain(endpoint),
							LastExecuted:    time.Time{},
							MinimumInterval: time.Duration(check.MinimumInterval) * time.Second,
						}
						cm.checkQueue.Add(item)
					}
				}
			}
		}
	}
}

func (cm *CheckManager) maintainQueue() {
	defer cm.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Reload configuration and update checks if needed
			cm.updateChecksFromConfig()
		case <-cm.shutdownCh:
			return
		}
	}
}

func (cm *CheckManager) updateChecksFromConfig() {
	// This could be enhanced to detect config changes and add/remove checks
	// For now, it's a placeholder for future enhancement
}

func (w *Worker) run() {
	defer w.manager.wg.Done()

	// Initial delay to stagger workers
	time.Sleep(w.startDelay)

	// Calculate interval: numWorkers * separationMs
	interval := time.Duration(w.manager.numWorkers*w.manager.separationMs) * time.Millisecond
	w.ticker = time.NewTicker(interval)
	defer w.ticker.Stop()

	log.Log(log.Debug, "Worker %d started with %v interval after %v delay",
		w.id, interval, w.startDelay)

	// Execute first check immediately after delay
	w.executeNextCheck()

	for {
		select {
		case <-w.ticker.C:
			w.executeNextCheck()
		case <-w.manager.shutdownCh:
			return
		}
	}
}

func (w *Worker) executeNextCheck() {
	item := w.manager.checkQueue.GetNext()
	if item == nil {
		return
	}

	// Execute the check
	w.executeCheck(item)

	// Update last executed time
	item.LastExecuted = time.Now()

	// Re-queue the check
	w.manager.checkQueue.Add(item)
}

func (w *Worker) executeCheck(item *CheckItem) {
	defer func() {
		if r := recover(); r != nil {
			log.Log(log.Error, "Worker %d: Check panic for %s/%s: %v",
				w.id, item.Check.Name, item.Member.Details.Name, r)
		}
	}()

	switch item.Type {
	case "site":
		if fn, ok := getSiteCheck(item.Check.Name); ok {
			fn(item.Check, item.Member)
		}
	case "domain":
		if fn, ok := getDomainCheck(item.Check.Name); ok {
			fn(item.Check, item.Domain, item.Service, item.Member)
		}
	case "endpoint":
		if fn, ok := getEndpointCheck(item.Check.Name); ok {
			fn(item.Check, item.Endpoint, item.Service, item.Member)
		}
	}
}
