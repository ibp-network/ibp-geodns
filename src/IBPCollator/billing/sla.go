package billing

// ──────────────────────────────────────────────────────────────────────────────
// Stake Plus Inc. – IBPCollator
// Billing subsystem – *minimal* SLA helper
//
// The full, DB‑backed SLA engine will be re‑introduced once the data‑layer
// work is finished.  For now we only need a stub that preserves the original
// call‑site signature (month UTC, *Summary) and always returns 100 % uptime,
// ensuring the project builds while we focus on the PDF visual overhaul.
// ──────────────────────────────────────────────────────────────────────────────

import "time"

// SLABreakdown captures the availability of a single <member,service> pair.
type SLABreakdown struct {
	HoursTotal int
	HoursDown  int
	HoursUp    int
	Uptime     float64 // 0–100
}

// SLASummary maps member → service → breakdown.
type SLASummary map[string]map[string]SLABreakdown

// CalculateSLAAdjustments keeps the original two‑parameter signature so that
// existing callers (in `billing.go`) compile unchanged.
//
// It currently returns a matrix with 100 % uptime.  This is **good enough for
// PDF generation** – the numbers appear, but no penalties are applied.
//
// Once the storage layer is wired in, we will replace this body with the
// fully‑featured version that queries downtime_log.
func CalculateSLAAdjustments(month time.Time, sum *Summary) (SLASummary, error) {
	out := make(SLASummary)

	for memberID, m := range sum.Members {
		if _, ok := out[memberID]; !ok {
			out[memberID] = make(map[string]SLABreakdown)
		}

		for svcKey := range m.ServiceCosts {
			out[memberID][svcKey] = SLABreakdown{
				HoursTotal: 0,
				HoursDown:  0,
				HoursUp:    0,
				Uptime:     100.0,
			}
		}
	}
	return out, nil
}
