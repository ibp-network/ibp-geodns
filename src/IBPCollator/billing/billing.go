package billing

// ─────────────────────────────────────────────────────────────────────────────
//  Stake Plus Inc. – IBP GeoDNS / IBPCollator – Billing subsystem
// ─────────────────────────────────────────────────────────────────────────────

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

// ─────────────────────────────────────────────────────────────────────────────
//  Public structures and accessors
// ─────────────────────────────────────────────────────────────────────────────

// MemberCost houses the breakdown of costs that *one* member incurs.
type MemberCost struct {
	MemberName   string
	ServiceCosts map[string]float64 // serviceName → $ cost
	Total        float64
}

// ServiceCost houses the breakdown of costs *per service* across all members.
type ServiceCost struct {
	ServiceName string
	MemberCosts map[string]float64 // memberName → $ cost
	Total       float64
}

// Summary keeps both perspectives together (no mutex — read-only snapshot).
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

// Track last generated billing month to avoid duplicates
var lastGeneratedBillingMonth time.Time
var billingGenMutex sync.Mutex

// GetSummary returns a deep-copy of the current billing snapshot.
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

// ─────────────────────────────────────────────────────────────────────────────
//  Initialisation
// ─────────────────────────────────────────────────────────────────────────────

// Init kicks off periodic billing refreshes and monthly billing PDF generation.
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

	// Daily service cost PDF generation at 00:05 UTC
	go func() {
		for {
			next := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour).Add(5 * time.Minute)
			time.Sleep(time.Until(next))
			generateServiceCostPDF()
		}
	}()

	// Monthly member billing PDF generation
	go func() {
		for {
			// Calculate next month's first day at 00:05 UTC
			now := time.Now().UTC()
			nextMonth := time.Date(now.Year(), now.Month(), 1, 0, 5, 0, 0, time.UTC).AddDate(0, 1, 0)

			// If we're already past the 5th minute of the first day, wait for next month
			if now.Day() == 1 && now.Hour() == 0 && now.Minute() >= 5 {
				nextMonth = nextMonth.AddDate(0, 1, 0)
			}

			waitDuration := time.Until(nextMonth)
			log.Log(log.Info, "[billing] Next member billing PDF generation scheduled for %s (in %v)",
				nextMonth.Format("2006-01-02 15:04:05"), waitDuration)

			time.Sleep(waitDuration)
			generateMonthlyBillingPDF()
		}
	}()

	// Generate initial PDFs if we haven't generated for the previous month yet
	go func() {
		time.Sleep(5 * time.Second) // Reduced delay since DB is now ready

		// Check if we need to generate last month's billing
		now := time.Now().UTC()
		if now.Day() >= 1 { // We're past the first of the month
			lastMonth := now.AddDate(0, -1, 0)
			lastMonthStart := time.Date(lastMonth.Year(), lastMonth.Month(), 1, 0, 0, 0, 0, time.UTC)

			billingGenMutex.Lock()
			needsGeneration := lastGeneratedBillingMonth.Before(lastMonthStart)
			billingGenMutex.Unlock()

			if needsGeneration {
				log.Log(log.Info, "[billing] Generating initial member billing PDF for previous month")
				generateMonthlyBillingPDF()
			}
		}

		// Always generate current service cost PDF
		generateServiceCostPDF()
	}()
}

// ─────────────────────────────────────────────────────────────────────────────
//  Refresh logic
// ─────────────────────────────────────────────────────────────────────────────

func refresh(verbose bool) {
	start := time.Now()
	c := cfg.GetConfig()

	newMemberCosts := make(map[string]MemberCost)
	newServiceCosts := make(map[string]ServiceCost)

	// indices (case-insensitive)
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
			log.Log(log.Warn, "[billing] region %q has no pricing entry — member %s skipped", mem.Location.Region, memName)
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
					log.Log(log.Warn, "[billing] unknown service %q referenced by member %s — skipped", svcName, memName)
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
	log.Log(log.Info, "[billing] refresh complete — %d members, %d services, in %s",
		len(newMemberCosts), len(newServiceCosts), duration)

	if verbose {
		logDetails(newMemberCosts, newServiceCosts)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
//  PDF Generation
// ─────────────────────────────────────────────────────────────────────────────

func generateServiceCostPDF() {
	conf := cfg.GetConfig()
	tmpDir := resolveTempDir(conf)
	if tmpDir == "" {
		log.Log(log.Warn, "[billing] tmp directory not configured — service cost PDF skipped")
		return
	}

	snap := GetSummary()
	if err := writeServiceCostPDF(&snap, tmpDir); err != nil {
		log.Log(log.Error, "[billing] failed to write service-cost PDF: %v", err)
	}
}

func generateMonthlyBillingPDF() {
	// Get the previous month
	now := time.Now().UTC()
	previousMonth := now.AddDate(0, -1, 0)
	billingMonth := time.Date(previousMonth.Year(), previousMonth.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Check if we've already generated for this month
	billingGenMutex.Lock()
	if !lastGeneratedBillingMonth.Before(billingMonth) {
		billingGenMutex.Unlock()
		log.Log(log.Info, "[billing] Member billing PDF already generated for %s", billingMonth.Format("January 2006"))
		return
	}
	lastGeneratedBillingMonth = billingMonth
	billingGenMutex.Unlock()

	log.Log(log.Info, "[billing] Starting member billing PDF generation for %s", billingMonth.Format("January 2006"))

	conf := cfg.GetConfig()
	tmpDir := resolveTempDir(conf)
	if tmpDir == "" {
		log.Log(log.Warn, "[billing] tmp directory not configured — member billing PDF skipped")
		return
	}

	// Create month directory (YYYY-MM format)
	monthDir := filepath.Join(tmpDir, billingMonth.Format("2006-01"))
	if err := os.MkdirAll(monthDir, 0755); err != nil {
		log.Log(log.Error, "[billing] Failed to create month directory: %v", err)
		return
	}

	snap := GetSummary()

	// Calculate SLA for the billing month
	sla, err := CalculateSLAAdjustments(billingMonth, &snap)
	if err != nil {
		log.Log(log.Error, "[billing] failed SLA calculation: %v", err)
		// Continue anyway with empty SLA data
		sla = make(SLASummary)
	}

	// Log members not meeting SLA
	violationCount := 0
	for memberName, services := range sla {
		for serviceName, breakdown := range services {
			if !breakdown.MeetsSLA {
				violationCount++
				log.Log(log.Warn, "[billing] SLA VIOLATION: %s / %s - Uptime: %.2f%% (Required: %.2f%%), Down: %.2f hrs",
					memberName, serviceName, breakdown.Uptime, breakdown.SLAThreshold, breakdown.HoursDown)
			}
		}
	}

	if violationCount == 0 {
		log.Log(log.Info, "[billing] No SLA violations detected for %s", billingMonth.Format("January 2006"))
	} else {
		log.Log(log.Info, "[billing] Total SLA violations for %s: %d", billingMonth.Format("January 2006"), violationCount)
	}

	// Generate the monthly overview PDF
	if err := writeMonthlyOverviewPDF(&snap, sla, monthDir, billingMonth); err != nil {
		log.Log(log.Error, "[billing] failed to write monthly overview PDF: %v", err)
	}

	// Generate individual member PDFs
	for memberName := range snap.Members {
		if err := writeMemberPDF(memberName, &snap, sla, monthDir, billingMonth); err != nil {
			log.Log(log.Error, "[billing] failed to write member PDF for %s: %v", memberName, err)
		}
	}

	log.Log(log.Info, "[billing] Monthly billing generation completed for %s", billingMonth.Format("January 2006"))
}

// ─────────────────────────────────────────────────────────────────────────────
//  Helpers
// ─────────────────────────────────────────────────────────────────────────────

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
		log.Log(log.Info, "[billing] %s — $%.2f", mc.MemberName, mc.Total)
		svcNames := make([]string, 0, len(mc.ServiceCosts))
		for s := range mc.ServiceCosts {
			svcNames = append(svcNames, s)
		}
		sort.Strings(svcNames)
		for _, s := range svcNames {
			log.Log(log.Info, "[billing]   • %s — $%.2f", s, mc.ServiceCosts[s])
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
		log.Log(log.Info, "[billing] %s — $%.2f", sc.ServiceName, sc.Total)
		memNames := make([]string, 0, len(sc.MemberCosts))
		for m := range sc.MemberCosts {
			memNames = append(memNames, m)
		}
		sort.Strings(memNames)
		for _, m := range memNames {
			log.Log(log.Info, "[billing]   • %s — $%.2f", m, sc.MemberCosts[m])
		}
	}
}

func resolveTempDir(conf interface{}) string {
	c := cfg.GetConfig()
	return filepath.Join(c.Local.System.WorkDir, "tmp")
}
