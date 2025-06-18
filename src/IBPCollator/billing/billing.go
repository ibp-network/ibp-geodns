package billing

// ──────────────────────────────────────────────────────────────────────────────
//  Stake Plus Inc. – IBP GeoDNS / IBPCollator – Billing subsystem
// ──────────────────────────────────────────────────────────────────────────────

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
)

// ──────────────────────────────────────────────────────────────────────────────
//  Public structures and accessors
// ──────────────────────────────────────────────────────────────────────────────

// MemberCost houses the breakdown of costs that *one* member incurs.
type MemberCost struct {
	MemberName   string
	ServiceCosts map[string]float64 // serviceName ➜ $ cost
	Total        float64
}

// ServiceCost houses the breakdown of costs *per service* across all members.
type ServiceCost struct {
	ServiceName string
	MemberCosts map[string]float64 // memberName ➜ $ cost
	Total       float64
}

// Summary keeps both perspectives together (no mutex – read‑only snapshot).
type Summary struct {
	Members  map[string]MemberCost
	Services map[string]ServiceCost
	Refresh  time.Time
}

// internal store guarded by a mutex
var billingStore struct {
	sync.RWMutex
	Summary
}

// GetSummary returns a deep‑copy of the current billing snapshot.
func GetSummary() Summary {
	billingStore.RLock()
	defer billingStore.RUnlock()

	// deep copy to ensure immutability
	mCopy := make(map[string]MemberCost, len(billingStore.Members))
	for k, v := range billingStore.Members {
		svcCopy := make(map[string]float64, len(v.ServiceCosts))
		for sk, sv := range v.ServiceCosts {
			svcCopy[sk] = sv
		}
		mCopy[k] = MemberCost{MemberName: v.MemberName, ServiceCosts: svcCopy, Total: v.Total}
	}
	sCopy := make(map[string]ServiceCost, len(billingStore.Services))
	for k, v := range billingStore.Services {
		memCopy := make(map[string]float64, len(v.MemberCosts))
		for mk, mv := range v.MemberCosts {
			memCopy[mk] = mv
		}
		sCopy[k] = ServiceCost{ServiceName: v.ServiceName, MemberCosts: memCopy, Total: v.Total}
	}

	return Summary{Members: mCopy, Services: sCopy, Refresh: billingStore.Refresh}
}

// ──────────────────────────────────────────────────────────────────────────────
//  Initialisation
// ──────────────────────────────────────────────────────────────────────────────

// Init kicks off periodic billing refreshes and daily PDF exports.
func Init() {
	// synchronous first refresh with verbose output
	refresh(true)

	// hourly refresh (top of the hour, UTC)
	go func() {
		for {
			next := time.Now().UTC().Truncate(time.Hour).Add(time.Hour)
			time.Sleep(time.Until(next))
			refresh(false)
		}
	}()

	// daily PDF generation at 00:05 UTC
	go func() {
		for {
			next := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour).Add(5 * time.Minute)
			time.Sleep(time.Until(next))
			runPDFExports()
		}
	}()

	// first set of PDFs right after start‑up
	runPDFExports()
}

// ──────────────────────────────────────────────────────────────────────────────
//  Refresh logic
// ──────────────────────────────────────────────────────────────────────────────

func refresh(verbose bool) {
	start := time.Now()
	c := cfg.GetConfig()

	newMemberCosts := make(map[string]MemberCost)
	newServiceCosts := make(map[string]ServiceCost)

	// indices (case‑insensitive)
	svcByName := make(map[string]cfg.Service)
	for n, s := range c.Services {
		svcByName[strings.ToLower(strings.TrimSpace(n))] = s
	}
	priceByRegion := make(map[string]cfg.IaasPricing)
	for r, p := range c.Pricing {
		priceByRegion[strings.ToLower(strings.TrimSpace(r))] = p
	}

	for memName, mem := range c.Members {
		regionKey := strings.ToLower(strings.TrimSpace(mem.Location.Region))
		price, ok := priceByRegion[regionKey]
		if !ok {
			log.Log(log.Warn, "[billing] region %q has no pricing entry – member %s skipped", mem.Location.Region, memName)
			continue
		}

		memCost := MemberCost{
			MemberName:   memName,
			ServiceCosts: map[string]float64{},
		}

		for _, svcList := range mem.ServiceAssignments {
			for _, svcName := range svcList {
				svc, exists := svcByName[strings.ToLower(strings.TrimSpace(svcName))]
				if !exists {
					log.Log(log.Warn, "[billing] unknown service %q referenced by member %s – skipped", svcName, memName)
					continue
				}

				cost := costForServiceInstance(svc.Resources, price)

				memCost.ServiceCosts[svcName] += cost
				memCost.Total += cost

				sc := newServiceCosts[svcName]
				if sc.ServiceName == "" {
					sc.ServiceName = svcName
					sc.MemberCosts = map[string]float64{}
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

	// publish atomically
	billingStore.Lock()
	billingStore.Members = newMemberCosts
	billingStore.Services = newServiceCosts
	billingStore.Refresh = time.Now().UTC()
	billingStore.Unlock()

	duration := time.Since(start).Round(time.Millisecond)
	log.Log(log.Info, "[billing] refresh complete – %d members, %d services, in %s",
		len(newMemberCosts), len(newServiceCosts), duration)

	if verbose {
		logDetails(newMemberCosts, newServiceCosts)
	}
}

// ──────────────────────────────────────────────────────────────────────────────
//  Helpers
// ──────────────────────────────────────────────────────────────────────────────

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

func logDetails(memCosts map[string]MemberCost, svcCosts map[string]ServiceCost) {
	log.Log(log.Info, "[billing] ---------------------- per member cost breakdown ----------------------")
	memberNames := make([]string, 0, len(memCosts))
	for n := range memCosts {
		memberNames = append(memberNames, n)
	}
	sort.Strings(memberNames)
	for _, m := range memberNames {
		mc := memCosts[m]
		log.Log(log.Info, "[billing] %s – $%.2f", mc.MemberName, mc.Total)
		svcNames := make([]string, 0, len(mc.ServiceCosts))
		for s := range mc.ServiceCosts {
			svcNames = append(svcNames, s)
		}
		sort.Strings(svcNames)
		for _, s := range svcNames {
			log.Log(log.Info, "[billing]   • %s – $%.2f", s, mc.ServiceCosts[s])
		}
	}

	log.Log(log.Info, "[billing] ---------------------- per service cost breakdown --------------------")
	serviceNames := make([]string, 0, len(svcCosts))
	for s := range svcCosts {
		serviceNames = append(serviceNames, s)
	}
	sort.Strings(serviceNames)
	for _, s := range serviceNames {
		sc := svcCosts[s]
		log.Log(log.Info, "[billing] %s – $%.2f", sc.ServiceName, sc.Total)
		memNames := make([]string, 0, len(sc.MemberCosts))
		for m := range sc.MemberCosts {
			memNames = append(memNames, m)
		}
		sort.Strings(memNames)
		for _, m := range memNames {
			log.Log(log.Info, "[billing]   • %s – $%.2f", m, sc.MemberCosts[m])
		}
	}
}

func runPDFExports() {
	conf := cfg.GetConfig()
	tmpDir := resolveTempDir(conf)
	if tmpDir == "" {
		log.Log(log.Warn, "[billing] tmp directory not configured – PDF exports skipped")
		return
	}

	snap := GetSummary()

	if err := writeServiceCostPDF(&snap, tmpDir); err != nil {
		log.Log(log.Error, "[billing] failed to write service‑cost PDF: %v", err)
	}

	month := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	sla, err := CalculateSLAAdjustments(month, &snap)
	if err != nil {
		log.Log(log.Error, "[billing] failed SLA calculation: %v", err)
		return
	}
	if err := writeMemberBillingPDF(&snap, sla, tmpDir, month); err != nil {
		log.Log(log.Error, "[billing] failed to write member‑billing PDF: %v", err)
	}
}

func resolveTempDir(conf interface{}) string {
	c := cfg.GetConfig()
	return filepath.Join(c.Local.System.WorkDir, "tmp")
}

// ensureDir verifies that path exists (creates it) and returns canonical version.
func ensureDir(p string) string {
	if p == "" {
		return ""
	}
	if err := os.MkdirAll(p, 0o755); err != nil {
		log.Log(log.Warn, "[billing] unable to create tmp dir %s: %v", p, err)
		return ""
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}
