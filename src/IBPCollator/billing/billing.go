package billing

// ──────────────────────────────────────────────────────────────────────────────
//  Stake Plus Inc. – IBP GeoDNS / IBPCollator – Billing subsystem
// ──────────────────────────────────────────────────────────────────────────────
//
//  This package calculates the infrastructure reimbursement owed to each
//  member (validator / RPC‑provider) and, conversely, the total cost of running
//  each distinct service configuration.
//
//  •  Per‑member view   → what each member earns for the services it hosts
//  •  Per‑service view  → aggregate cost of all members that host that service
//
//  Calculation model
//  ─────────────────
//  1.  Member ➜ Region ➜ IaaS pricing table
//  2.  Service ➜ Resources (nodes, cores, memory, disk, bandwidth)
//  3.  Every member runs one *instance* of every service it has been assigned.
//  4.  Cost for a single node = Σ(componentQty × componentUnitPrice)
//     Cost for the service    = costPerNode × nodes
//
//  Results are stored in memory (thread‑safe) and refreshed once per hour UTC.
// ──────────────────────────────────────────────────────────────────────────────

import (
	"strings"
	"sync"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
)

// ──────────────────────────────────────────────────────────────────────────────
//  Public structures and accessors
// ──────────────────────────────────────────────────────────────────────────────

// MemberCost houses the breakdown of costs that *one* member incurs by hosting
// services.
type MemberCost struct {
	MemberName   string
	ServiceCosts map[string]float64 // serviceName ➜ dollar cost
	Total        float64
}

// ServiceCost houses the breakdown of costs *per service* across all members.
type ServiceCost struct {
	ServiceName string
	MemberCosts map[string]float64 // memberName ➜ dollar cost
	Total       float64
}

// Summary keeps both perspectives together.
type Summary struct {
	Members  map[string]MemberCost
	Services map[string]ServiceCost
	Refresh  time.Time
	mu       sync.RWMutex
}

// singleton instance
var billingSummary Summary

// GetSummary returns a deep‑copy of the current billing snapshot.
func GetSummary() Summary {
	billingSummary.mu.RLock()
	defer billingSummary.mu.RUnlock()

	// deep‑copy maps to avoid data races
	mCopy := make(map[string]MemberCost, len(billingSummary.Members))
	for k, v := range billingSummary.Members {
		svcCopy := make(map[string]float64, len(v.ServiceCosts))
		for sk, sv := range v.ServiceCosts {
			svcCopy[sk] = sv
		}
		mCopy[k] = MemberCost{
			MemberName:   v.MemberName,
			ServiceCosts: svcCopy,
			Total:        v.Total,
		}
	}

	sCopy := make(map[string]ServiceCost, len(billingSummary.Services))
	for k, v := range billingSummary.Services {
		memCopy := make(map[string]float64, len(v.MemberCosts))
		for mk, mv := range v.MemberCosts {
			memCopy[mk] = mv
		}
		sCopy[k] = ServiceCost{
			ServiceName: v.ServiceName,
			MemberCosts: memCopy,
			Total:       v.Total,
		}
	}

	return Summary{
		Members:  mCopy,
		Services: sCopy,
		Refresh:  billingSummary.Refresh,
	}
}

// ──────────────────────────────────────────────────────────────────────────────
//  Initialisation
// ──────────────────────────────────────────────────────────────────────────────

// Init must be called once from main(); it kicks off an hourly refresh job.
func Init() {
	// first run immediately
	refreshOnce()

	// schedule hourly on the top of the hour (UTC)
	go func() {
		for {
			now := time.Now().UTC()
			next := now.Truncate(time.Hour).Add(time.Hour)
			time.Sleep(time.Until(next))
			refreshOnce()
		}
	}()
}

// ──────────────────────────────────────────────────────────────────────────────
//  Internal helpers
// ──────────────────────────────────────────────────────────────────────────────

// refreshOnce recalculates the entire billing table atomically.
func refreshOnce() {
	start := time.Now()
	c := cfg.GetConfig()

	newMemberCosts := make(map[string]MemberCost)
	newServiceCosts := make(map[string]ServiceCost)

	// pre‑index services for quick lookup
	svcByName := make(map[string]cfg.Service)
	for name, svc := range c.Services {
		svcByName[name] = svc
	}

	for memName, mem := range c.Members {
		region := strings.ToLower(strings.TrimSpace(mem.Location.Region))
		price, ok := c.Pricing[region]
		if !ok {
			log.Log(log.Warn, "[billing] region %q has no pricing entry – member %s skipped", region, memName)
			continue
		}

		memCost := MemberCost{
			MemberName:   memName,
			ServiceCosts: make(map[string]float64),
		}

		// ServiceAssignments is map[string][]string; we don't care about the key,
		// only the service names inside the slices.
		for _, svcList := range mem.ServiceAssignments {
			for _, svcName := range svcList {
				svc, svcOK := svcByName[svcName]
				if !svcOK {
					log.Log(log.Warn, "[billing] unknown service %q referenced by member %s – skipped", svcName, memName)
					continue
				}

				cost := costForServiceInstance(svc.Resources, price)

				// update per‑member view
				memCost.ServiceCosts[svcName] += cost
				memCost.Total += cost

				// update per‑service view
				sc := newServiceCosts[svcName]
				if sc.ServiceName == "" {
					sc.ServiceName = svcName
					sc.MemberCosts = make(map[string]float64)
				}
				sc.MemberCosts[memName] += cost
				sc.Total += cost
				newServiceCosts[svcName] = sc
			}
		}

		if memCost.Total > 0 {
			newMemberCosts[memName] = memCost
		}
	}

	billingSummary.mu.Lock()
	billingSummary.Members = newMemberCosts
	billingSummary.Services = newServiceCosts
	billingSummary.Refresh = time.Now().UTC()
	billingSummary.mu.Unlock()

	duration := time.Since(start)
	log.Log(log.Info, "[billing] refresh complete – %d members, %d services, in %s",
		len(newMemberCosts), len(newServiceCosts), duration.Round(time.Millisecond))
}

// costForServiceInstance returns the total USD cost of *one instance* of a
// service, given the resources and regional IaaS pricing.
func costForServiceInstance(res cfg.Resources, price cfg.IaasPricing) float64 {
	if res.Nodes == 0 {
		return 0
	}

	perNode := (float64(res.Cores) * price.Cores) +
		(float64(res.Memory) * price.Memory) +
		(float64(res.Disk) * price.Disk) +
		(float64(res.Bandwidth) * price.Bandwidth)

	return perNode * float64(res.Nodes)
}
