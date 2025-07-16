package billing

import (
	"fmt"
	"path/filepath"
	"sort"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"

	"github.com/phpdave11/gofpdf"
)

// writeMonthlyOverviewPDF generates a summary PDF for all members
func writeMonthlyOverviewPDF(sum *Summary, sla SLASummary, tmpDir string, month time.Time) error {
	baseDir := filepath.Dir(tmpDir)
	logoPath := findLogo(baseDir)

	const titleFmt = "IBP Network - Monthly Overview (%s %d)"
	title := fmt.Sprintf(titleFmt, month.Format("January"), month.Year())

	pdf := gofpdf.New("L", "mm", "A4", "") // Landscape orientation for more columns
	pdf.SetTitle(title, false)
	pdf.SetAuthor("IBPCollator "+Version(), false)

	// Global header
	pdf.SetHeaderFuncMode(func() {
		pdf.SetFont("Helvetica", "B", 15)
		pdf.CellFormat(0, 10, title, "", 1, "C", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.CellFormat(0, 6, time.Now().UTC().Format("02 Jan 2006 15:04 UTC"),
			"", 0, "C", false, 0, "")
	}, true)

	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont("Helvetica", "I", 9)
		pdf.CellFormat(0, 10,
			fmt.Sprintf("page %d of {nb}", pdf.PageNo()), "", 0, "C", false, 0, "")
	})

	pdf.AliasNbPages("")
	addPageWithWatermark(pdf, logoPath)

	// Calculate statistics
	memberStats := calculateMemberStats(month)
	totalRequests := 0
	for _, stats := range memberStats {
		totalRequests += stats.RequestCount
	}

	// Get member names in order
	memberNames := make([]string, 0, len(sum.Members))
	for m := range sum.Members {
		memberNames = append(memberNames, m)
	}
	sort.Strings(memberNames)

	// Summary statistics at the top
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(0, 8, fmt.Sprintf("Total DNS Requests: %d", totalRequests), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 8, fmt.Sprintf("Active Members: %d", len(memberNames)), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 8, fmt.Sprintf("Billing Period: %s", month.Format("January 2006")), "", 1, "L", false, 0, "")
	pdf.Ln(5)

	// Table columns
	const (
		colMemberW   = 40.0
		colLevelW    = 15.0
		colRequestsW = 30.0
		colPercentW  = 25.0
		colServicesW = 20.0
		colDowntimeW = 30.0
		colBaseCostW = 30.0
		colBilledW   = 30.0
		colUptimeW   = 25.0
		rowH         = 7.0
	)

	// Calculate totals
	var grandTotalBase, grandTotalBilled float64
	totalDowntimeServices := 0

	// Prepare data
	type memberRow struct {
		name             string
		level            int
		requests         int
		percentage       float64
		serviceCount     int
		downtimeServices int
		baseCost         float64
		billedCost       float64
		avgUptime        float64
		meetsSLA         bool
	}

	memberData := make([]memberRow, 0, len(memberNames))
	c := cfg.GetConfig()

	for _, mem := range memberNames {
		row := memberRow{name: mem}

		// Get member config
		if memberConfig, exists := c.Members[mem]; exists {
			row.level = memberConfig.Membership.Level
		}

		// Get request stats
		if stats, exists := memberStats[mem]; exists {
			row.requests = stats.RequestCount
			if totalRequests > 0 {
				row.percentage = float64(stats.RequestCount) / float64(totalRequests) * 100.0
			}
		}

		// Calculate costs and uptime
		row.serviceCount = len(sum.Members[mem].ServiceCosts)
		totalUptime := 0.0
		uptimeCount := 0

		for svcName, baseCost := range sum.Members[mem].ServiceCosts {
			row.baseCost += baseCost
			breakdown := getSLABreakdown(sla, mem, svcName)

			if breakdown.HoursDown > 0 {
				row.downtimeServices++
			}

			if !breakdown.MeetsSLA {
				row.meetsSLA = false
			}

			totalUptime += breakdown.Uptime
			uptimeCount++

			billed := baseCost * (breakdown.Uptime / 100.0)
			row.billedCost += billed
		}

		if uptimeCount > 0 {
			row.avgUptime = totalUptime / float64(uptimeCount)
		} else {
			row.avgUptime = 100.0
		}

		memberData = append(memberData, row)
		grandTotalBase += row.baseCost
		grandTotalBilled += row.billedCost
		totalDowntimeServices += row.downtimeServices
	}

	// Table header
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetFillColor(240, 240, 240)

	pdf.CellFormat(colMemberW, rowH, "Member", "1", 0, "L", true, 0, "")
	pdf.CellFormat(colLevelW, rowH, "Level", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colRequestsW, rowH, "Requests", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colPercentW, rowH, "% of Total", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colServicesW, rowH, "Services", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colDowntimeW, rowH, "Down Services", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colUptimeW, rowH, "Avg Uptime", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colBaseCostW, rowH, "Base Cost", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colBilledW, rowH, "Billed Amount", "1", 1, "R", true, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	fillToggle := false

	// Table rows
	for _, row := range memberData {
		if pdf.GetY() > 180 { // Landscape has less vertical space
			addPageWithWatermark(pdf, logoPath)

			// Reprint header
			pdf.SetFont("Helvetica", "B", 10)
			pdf.SetFillColor(240, 240, 240)

			pdf.CellFormat(colMemberW, rowH, "Member", "1", 0, "L", true, 0, "")
			pdf.CellFormat(colLevelW, rowH, "Level", "1", 0, "C", true, 0, "")
			pdf.CellFormat(colRequestsW, rowH, "Requests", "1", 0, "R", true, 0, "")
			pdf.CellFormat(colPercentW, rowH, "% of Total", "1", 0, "R", true, 0, "")
			pdf.CellFormat(colServicesW, rowH, "Services", "1", 0, "C", true, 0, "")
			pdf.CellFormat(colDowntimeW, rowH, "Down Services", "1", 0, "C", true, 0, "")
			pdf.CellFormat(colUptimeW, rowH, "Avg Uptime", "1", 0, "R", true, 0, "")
			pdf.CellFormat(colBaseCostW, rowH, "Base Cost", "1", 0, "R", true, 0, "")
			pdf.CellFormat(colBilledW, rowH, "Billed Amount", "1", 1, "R", true, 0, "")

			pdf.SetFont("Helvetica", "", 9)
		}

		fillToggle = !fillToggle

		// Member name
		pdf.CellFormat(colMemberW, rowH, row.name, "1", 0, "L", fillToggle, 0, "")

		// Level
		pdf.CellFormat(colLevelW, rowH, fmt.Sprintf("%d", row.level), "1", 0, "C", fillToggle, 0, "")

		// Requests
		pdf.CellFormat(colRequestsW, rowH, fmt.Sprintf("%d", row.requests), "1", 0, "R", fillToggle, 0, "")

		// Percentage
		pdf.CellFormat(colPercentW, rowH, fmt.Sprintf("%.2f%%", row.percentage), "1", 0, "R", fillToggle, 0, "")

		// Services
		pdf.CellFormat(colServicesW, rowH, fmt.Sprintf("%d", row.serviceCount), "1", 0, "C", fillToggle, 0, "")

		// Downtime services
		if row.downtimeServices > 0 {
			pdf.SetTextColor(255, 0, 0)
		}
		pdf.CellFormat(colDowntimeW, rowH, fmt.Sprintf("%d", row.downtimeServices), "1", 0, "C", fillToggle, 0, "")
		pdf.SetTextColor(0, 0, 0)

		// Average uptime
		if row.avgUptime < DefaultSLAPercentage {
			pdf.SetTextColor(255, 0, 0)
		}
		pdf.CellFormat(colUptimeW, rowH, fmt.Sprintf("%.2f%%", row.avgUptime), "1", 0, "R", fillToggle, 0, "")
		pdf.SetTextColor(0, 0, 0)

		// Base cost
		pdf.CellFormat(colBaseCostW, rowH, fmt.Sprintf("$%.2f", row.baseCost), "1", 0, "R", fillToggle, 0, "")

		// Billed amount
		pdf.CellFormat(colBilledW, rowH, fmt.Sprintf("$%.2f", row.billedCost), "1", 1, "R", fillToggle, 0, "")
	}

	// Totals row
	pdf.SetFont("Helvetica", "B", 10)
	totalColWidth := colMemberW + colLevelW + colRequestsW + colPercentW + colServicesW + colDowntimeW + colUptimeW
	pdf.CellFormat(totalColWidth, rowH, "TOTALS", "1", 0, "R", false, 0, "")
	pdf.CellFormat(colBaseCostW, rowH, fmt.Sprintf("$%.2f", grandTotalBase), "1", 0, "R", false, 0, "")
	pdf.CellFormat(colBilledW, rowH, fmt.Sprintf("$%.2f", grandTotalBilled), "1", 1, "R", false, 0, "")

	// Summary section
	pdf.Ln(10)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(0, 8, "Summary Statistics", "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("Total Base Cost: $%.2f", grandTotalBase), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Total Billed Amount: $%.2f", grandTotalBilled), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Total Savings from SLA Credits: $%.2f", grandTotalBase-grandTotalBilled), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 6, fmt.Sprintf("Services with Downtime: %d", totalDowntimeServices), "", 1, "L", false, 0, "")

	// SLA violations section
	pdf.Ln(5)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(0, 8, "SLA Violations", "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	violationCount := 0
	for _, row := range memberData {
		if row.avgUptime < DefaultSLAPercentage {
			violationCount++
			pdf.SetTextColor(255, 0, 0)
			pdf.CellFormat(0, 5, fmt.Sprintf("• %s - Average uptime: %.2f%% (Required: %.2f%%)",
				row.name, row.avgUptime, DefaultSLAPercentage), "", 1, "L", false, 0, "")
			pdf.SetTextColor(0, 0, 0)
		}
	}

	if violationCount == 0 {
		pdf.SetTextColor(0, 128, 0)
		pdf.CellFormat(0, 5, "No SLA violations this month", "", 1, "L", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	}

	filename := filepath.Join(tmpDir,
		fmt.Sprintf("monthly_overview_%s.pdf", month.Format("200601")))
	if err := pdf.OutputFileAndClose(filename); err != nil {
		return err
	}

	log.Log(log.Info, "[billing] monthly overview PDF written → %s", filename)
	return nil
}
