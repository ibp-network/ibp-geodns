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

// writeMonthlyOverviewPDF generates a summary PDF for all members with modern design
func writeMonthlyOverviewPDF(sum *Summary, sla SLASummary, outDir string, month time.Time) error {
	logoPath := findLogo(filepath.Dir(outDir))
	filename := filepath.Join(outDir, fmt.Sprintf("%s-Monthly_Overview.pdf", month.Format("2006_01")))

	pdf := gofpdf.New("L", "mm", "A4", "") // Landscape
	pdf.SetTitle("IBP Network Monthly Overview", false)
	pdf.SetAuthor("IBPCollator "+Version(), false)

	// Modern header
	pdf.SetHeaderFuncMode(func() {
		pdf.SetFillColor(30, 30, 30)
		pdf.Rect(0, 0, 297, 25, "F")

		if logoPath != "" {
			pdf.Image(logoPath, 10, 5, 30, 0, false, "", 0, "")
		}

		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Helvetica", "B", 18)
		pdf.SetXY(50, 8)
		pdf.CellFormat(200, 8, "IBP Network Monthly Overview", "", 0, "L", false, 0, "")

		pdf.SetFont("Helvetica", "", 12)
		pdf.SetXY(50, 16)
		pdf.CellFormat(200, 5, month.Format("January 2006"), "", 0, "L", false, 0, "")

		pdf.SetTextColor(0, 0, 0)
		pdf.SetY(30)
	}, true)

	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont("Helvetica", "I", 8)
		pdf.SetTextColor(128, 128, 128)
		pdf.CellFormat(0, 10, fmt.Sprintf("Page %d of {nb}", pdf.PageNo()), "", 0, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	})

	pdf.AliasNbPages("")
	pdf.AddPage()

	// Calculate statistics
	memberStats := calculateMemberStats(month)
	totalRequests := calculateTotalRequests(month)

	// Get member names in order
	memberNames := make([]string, 0, len(sum.Members))
	for m := range sum.Members {
		memberNames = append(memberNames, m)
	}
	sort.Strings(memberNames)

	// Summary cards at the top
	y := 35.0
	cardWidth := 90.0
	cardHeight := 25.0
	spacing := 5.0

	// Total requests card
	drawGradientCard(pdf, 10, y, cardWidth, cardHeight, 70, 130, 180)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetXY(12, y+5)
	pdf.CellFormat(cardWidth-4, 5, "Total DNS Requests", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetXY(12, y+12)
	pdf.CellFormat(cardWidth-4, 8, formatNumber(totalRequests), "", 0, "C", false, 0, "")

	// Active members card
	drawGradientCard(pdf, 10+cardWidth+spacing, y, cardWidth, cardHeight, 46, 125, 50)
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetXY(12+cardWidth+spacing, y+5)
	pdf.CellFormat(cardWidth-4, 5, "Active Members", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetXY(12+cardWidth+spacing, y+12)
	pdf.CellFormat(cardWidth-4, 8, fmt.Sprintf("%d", len(memberNames)), "", 0, "C", false, 0, "")

	// Total billed card
	var grandTotalBilled float64
	for _, mem := range memberNames {
		for svcName, baseCost := range sum.Members[mem].ServiceCosts {
			breakdown := getSLABreakdown(sla, mem, svcName)
			grandTotalBilled += baseCost * (breakdown.Uptime / 100.0)
		}
	}

	drawGradientCard(pdf, 10+2*(cardWidth+spacing), y, cardWidth, cardHeight, 255, 152, 0)
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetXY(12+2*(cardWidth+spacing), y+5)
	pdf.CellFormat(cardWidth-4, 5, "Total Billed", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetXY(12+2*(cardWidth+spacing), y+12)
	pdf.CellFormat(cardWidth-4, 8, fmt.Sprintf("$%s", formatNumber(int(grandTotalBilled))), "", 0, "C", false, 0, "")

	pdf.SetTextColor(0, 0, 0)

	// Main table
	y = 70
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetXY(10, y)
	pdf.CellFormat(277, 8, "Member Performance Summary", "", 1, "L", false, 0, "")
	y += 10

	// Table setup
	const (
		colMemberW   = 45.0
		colLevelW    = 15.0
		colRequestsW = 35.0
		colPercentW  = 25.0
		colServicesW = 20.0
		colDowntimeW = 25.0
		colUptimeW   = 25.0
		colBaseCostW = 35.0
		colBilledW   = 35.0
		colStatusW   = 20.0
		rowH         = 8.0
	)

	// Table header with modern style
	pdf.SetFillColor(50, 50, 50)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 9)

	pdf.SetXY(10, y)
	pdf.CellFormat(colMemberW, rowH, "Member", "1", 0, "L", true, 0, "")
	pdf.CellFormat(colLevelW, rowH, "Lvl", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colRequestsW, rowH, "Requests", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colPercentW, rowH, "Share", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colServicesW, rowH, "Svcs", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colDowntimeW, rowH, "Down", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colUptimeW, rowH, "Uptime", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colBaseCostW, rowH, "Base Cost", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colBilledW, rowH, "Billed", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colStatusW, rowH, "SLA", "1", 1, "C", true, 0, "")

	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Helvetica", "", 9)
	y += rowH

	// Prepare member data
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

		if memberConfig, exists := c.Members[mem]; exists {
			row.level = memberConfig.Membership.Level
		}

		if stats, exists := memberStats[mem]; exists {
			row.requests = stats.RequestCount
			if totalRequests > 0 {
				row.percentage = float64(stats.RequestCount) / float64(totalRequests) * 100.0
			}
		}

		row.serviceCount = len(sum.Members[mem].ServiceCosts)
		totalUptime := 0.0
		uptimeCount := 0
		row.meetsSLA = true

		for svcName, baseCost := range sum.Members[mem].ServiceCosts {
			row.baseCost += baseCost
			breakdown := getSLABreakdown(sla, mem, svcName)

			// For site checks, if there's downtime it affects all services
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
	}

	// Table rows with alternating colors
	fillToggle := false
	for _, row := range memberData {
		if y > 180 {
			pdf.AddPage()
			y = 35

			// Reprint header
			pdf.SetFillColor(50, 50, 50)
			pdf.SetTextColor(255, 255, 255)
			pdf.SetFont("Helvetica", "B", 9)

			pdf.SetXY(10, y)
			pdf.CellFormat(colMemberW, rowH, "Member", "1", 0, "L", true, 0, "")
			pdf.CellFormat(colLevelW, rowH, "Lvl", "1", 0, "C", true, 0, "")
			pdf.CellFormat(colRequestsW, rowH, "Requests", "1", 0, "R", true, 0, "")
			pdf.CellFormat(colPercentW, rowH, "Share", "1", 0, "R", true, 0, "")
			pdf.CellFormat(colServicesW, rowH, "Svcs", "1", 0, "C", true, 0, "")
			pdf.CellFormat(colDowntimeW, rowH, "Down", "1", 0, "C", true, 0, "")
			pdf.CellFormat(colUptimeW, rowH, "Uptime", "1", 0, "R", true, 0, "")
			pdf.CellFormat(colBaseCostW, rowH, "Base Cost", "1", 0, "R", true, 0, "")
			pdf.CellFormat(colBilledW, rowH, "Billed", "1", 0, "R", true, 0, "")
			pdf.CellFormat(colStatusW, rowH, "SLA", "1", 1, "C", true, 0, "")

			pdf.SetTextColor(0, 0, 0)
			pdf.SetFont("Helvetica", "", 9)
			y += rowH
		}

		fillToggle = !fillToggle
		if fillToggle {
			pdf.SetFillColor(245, 245, 245)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		pdf.SetXY(10, y)

		// Member name
		pdf.CellFormat(colMemberW, rowH, row.name, "1", 0, "L", true, 0, "")

		// Level
		pdf.CellFormat(colLevelW, rowH, fmt.Sprintf("%d", row.level), "1", 0, "C", true, 0, "")

		// Requests
		pdf.CellFormat(colRequestsW, rowH, formatNumber(row.requests), "1", 0, "R", true, 0, "")

		// Percentage
		pdf.CellFormat(colPercentW, rowH, fmt.Sprintf("%.1f%%", row.percentage), "1", 0, "R", true, 0, "")

		// Services
		pdf.CellFormat(colServicesW, rowH, fmt.Sprintf("%d", row.serviceCount), "1", 0, "C", true, 0, "")

		// Downtime services
		if row.downtimeServices > 0 {
			pdf.SetTextColor(255, 0, 0)
		}
		pdf.CellFormat(colDowntimeW, rowH, fmt.Sprintf("%d", row.downtimeServices), "1", 0, "C", true, 0, "")
		pdf.SetTextColor(0, 0, 0)

		// Average uptime
		if row.avgUptime < DefaultSLAPercentage {
			pdf.SetTextColor(255, 0, 0)
		}
		pdf.CellFormat(colUptimeW, rowH, fmt.Sprintf("%.2f%%", row.avgUptime), "1", 0, "R", true, 0, "")
		pdf.SetTextColor(0, 0, 0)

		// Base cost
		pdf.CellFormat(colBaseCostW, rowH, fmt.Sprintf("$%.2f", row.baseCost), "1", 0, "R", true, 0, "")

		// Billed amount
		pdf.CellFormat(colBilledW, rowH, fmt.Sprintf("$%.2f", row.billedCost), "1", 0, "R", true, 0, "")

		// SLA status
		if row.meetsSLA {
			pdf.SetTextColor(0, 150, 0)
			pdf.CellFormat(colStatusW, rowH, "✓", "1", 1, "C", true, 0, "")
		} else {
			pdf.SetTextColor(255, 0, 0)
			pdf.CellFormat(colStatusW, rowH, "✗", "1", 1, "C", true, 0, "")
		}
		pdf.SetTextColor(0, 0, 0)

		y += rowH
	}

	if err := pdf.OutputFileAndClose(filename); err != nil {
		return err
	}

	log.Log(log.Info, "[billing] Monthly overview PDF written → %s", filename)
	return nil
}

// Helper functions for better design
func drawGradientCard(pdf *gofpdf.Fpdf, x, y, w, h float64, r, g, b int) {
	// Simple solid color card with shadow effect
	pdf.SetFillColor(r-20, g-20, b-20)
	pdf.Rect(x+1, y+1, w, h, "F")
	pdf.SetFillColor(r, g, b)
	pdf.Rect(x, y, w, h, "F")
}

func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1000000)
}

// calculateTotalRequests gets the total requests for the month
func calculateTotalRequests(month time.Time) int {
	stats := calculateMemberStats(month)
	total := 0
	for _, s := range stats {
		total += s.RequestCount
	}
	return total
}
