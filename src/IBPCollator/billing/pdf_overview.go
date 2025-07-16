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

	// Modern header with logo
	pdf.SetHeaderFuncMode(func() {
		// Dark header background
		pdf.SetFillColor(30, 30, 30)
		pdf.Rect(0, 0, 297, 30, "F")

		// Logo
		if logoPath != "" {
			pdf.Image(logoPath, 10, 7, 25, 0, false, "", 0, "")
		}

		// Title text
		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Helvetica", "B", 20)
		pdf.SetXY(45, 10)
		pdf.CellFormat(200, 10, "IBP Network Monthly Overview", "", 0, "L", false, 0, "")

		pdf.SetFont("Helvetica", "", 14)
		pdf.SetXY(45, 20)
		pdf.CellFormat(200, 6, month.Format("January 2006"), "", 0, "L", false, 0, "")

		// Date on right
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetXY(240, 15)
		pdf.CellFormat(50, 5, time.Now().UTC().Format("Generated: Jan 2, 2006"), "", 0, "R", false, 0, "")

		pdf.SetTextColor(0, 0, 0)
		pdf.SetY(35)
	}, true)

	pdf.SetFooterFunc(func() {
		pdf.SetY(-15)
		pdf.SetFont("Helvetica", "I", 8)
		pdf.SetTextColor(128, 128, 128)
		pdf.CellFormat(0, 10, fmt.Sprintf("Page %d of {nb}", pdf.PageNo()), "", 0, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	})

	pdf.AliasNbPages("")

	// Calculate all statistics first
	memberStats := calculateMemberStats(month)
	totalRequests := calculateTotalRequests(month)
	memberNames := make([]string, 0, len(sum.Members))
	for m := range sum.Members {
		memberNames = append(memberNames, m)
	}
	sort.Strings(memberNames)

	// Calculate totals
	var grandTotalBase, grandTotalBilled float64
	totalDowntimeServices := 0
	totalSLAViolations := 0
	avgNetworkUptime := 0.0

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

			if breakdown.HoursDown > 0 {
				row.downtimeServices++
			}

			if !breakdown.MeetsSLA {
				row.meetsSLA = false
				totalSLAViolations++
			}

			totalUptime += breakdown.Uptime
			uptimeCount++

			billed := baseCost * (breakdown.Uptime / 100.0)
			row.billedCost += billed
		}

		if uptimeCount > 0 {
			row.avgUptime = totalUptime / float64(uptimeCount)
			avgNetworkUptime += row.avgUptime
		} else {
			row.avgUptime = 100.0
			avgNetworkUptime += 100.0
		}

		memberData = append(memberData, row)
		grandTotalBase += row.baseCost
		grandTotalBilled += row.billedCost
		totalDowntimeServices += row.downtimeServices
	}

	if len(memberData) > 0 {
		avgNetworkUptime = avgNetworkUptime / float64(len(memberData))
	}

	// ===== PAGE 1: OVERVIEW =====
	pdf.AddPage()

	// Large logo watermark in background
	if logoPath != "" {
		pdf.SetAlpha(0.1, "Normal")
		pdf.Image(logoPath, 100, 60, 100, 0, false, "", 0, "")
		pdf.SetAlpha(1, "Normal")
	}

	// Network Statistics Section
	y := 45.0
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetXY(20, y)
	pdf.CellFormat(257, 10, "Network Performance Summary", "", 1, "L", false, 0, "")
	y += 15

	// Summary cards
	cardWidth := 80.0
	cardHeight := 35.0
	spacing := 10.0
	startX := 20.0

	// Card 1: Total Requests
	drawGradientCard(pdf, startX, y, cardWidth, cardHeight, 70, 130, 180)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "", 11)
	pdf.SetXY(startX+2, y+5)
	pdf.CellFormat(cardWidth-4, 6, "Total DNS Requests", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 20)
	pdf.SetXY(startX+2, y+15)
	pdf.CellFormat(cardWidth-4, 10, formatNumber(totalRequests), "", 0, "C", false, 0, "")

	// Card 2: Active Members
	drawGradientCard(pdf, startX+cardWidth+spacing, y, cardWidth, cardHeight, 46, 125, 50)
	pdf.SetFont("Helvetica", "", 11)
	pdf.SetXY(startX+cardWidth+spacing+2, y+5)
	pdf.CellFormat(cardWidth-4, 6, "Active Members", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 20)
	pdf.SetXY(startX+cardWidth+spacing+2, y+15)
	pdf.CellFormat(cardWidth-4, 10, fmt.Sprintf("%d", len(memberNames)), "", 0, "C", false, 0, "")

	// Card 3: Network Uptime
	uptimeColor := []int{255, 152, 0}
	if avgNetworkUptime >= DefaultSLAPercentage {
		uptimeColor = []int{46, 125, 50}
	}
	drawGradientCard(pdf, startX+2*(cardWidth+spacing), y, cardWidth, cardHeight, uptimeColor[0], uptimeColor[1], uptimeColor[2])
	pdf.SetFont("Helvetica", "", 11)
	pdf.SetXY(startX+2*(cardWidth+spacing)+2, y+5)
	pdf.CellFormat(cardWidth-4, 6, "Average Uptime", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 20)
	pdf.SetXY(startX+2*(cardWidth+spacing)+2, y+15)
	pdf.CellFormat(cardWidth-4, 10, fmt.Sprintf("%.2f%%", avgNetworkUptime), "", 0, "C", false, 0, "")

	pdf.SetTextColor(0, 0, 0)

	// Financial Summary
	y += 50
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetXY(20, y)
	pdf.CellFormat(257, 10, "Financial Summary", "", 1, "L", false, 0, "")
	y += 15

	// Financial cards
	cardHeight = 40.0

	// Base Cost Card
	drawGradientCard(pdf, startX, y, cardWidth, cardHeight, 100, 100, 100)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "", 11)
	pdf.SetXY(startX+2, y+5)
	pdf.CellFormat(cardWidth-4, 6, "Total Base Cost", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetXY(startX+2, y+15)
	pdf.CellFormat(cardWidth-4, 10, fmt.Sprintf("$%s", formatNumber(int(grandTotalBase))), "", 0, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetXY(startX+2, y+28)
	pdf.CellFormat(cardWidth-4, 5, "Before SLA adjustments", "", 0, "C", false, 0, "")

	// Billed Amount Card
	drawGradientCard(pdf, startX+cardWidth+spacing, y, cardWidth, cardHeight, 255, 152, 0)
	pdf.SetFont("Helvetica", "", 11)
	pdf.SetXY(startX+cardWidth+spacing+2, y+5)
	pdf.CellFormat(cardWidth-4, 6, "Total Billed", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetXY(startX+cardWidth+spacing+2, y+15)
	pdf.CellFormat(cardWidth-4, 10, fmt.Sprintf("$%s", formatNumber(int(grandTotalBilled))), "", 0, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetXY(startX+cardWidth+spacing+2, y+28)
	pdf.CellFormat(cardWidth-4, 5, "After SLA credits", "", 0, "C", false, 0, "")

	// SLA Credits Card
	savings := grandTotalBase - grandTotalBilled
	drawGradientCard(pdf, startX+2*(cardWidth+spacing), y, cardWidth, cardHeight, 46, 125, 50)
	pdf.SetFont("Helvetica", "", 11)
	pdf.SetXY(startX+2*(cardWidth+spacing)+2, y+5)
	pdf.CellFormat(cardWidth-4, 6, "SLA Credits", "", 1, "C", false, 0, "")
	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetXY(startX+2*(cardWidth+spacing)+2, y+15)
	pdf.CellFormat(cardWidth-4, 10, fmt.Sprintf("$%s", formatNumber(int(savings))), "", 0, "C", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetXY(startX+2*(cardWidth+spacing)+2, y+28)
	pdf.CellFormat(cardWidth-4, 5, fmt.Sprintf("%.1f%% savings", (savings/grandTotalBase)*100), "", 0, "C", false, 0, "")

	pdf.SetTextColor(0, 0, 0)

	// Service Health Summary
	y += 55
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetXY(20, y)
	pdf.CellFormat(257, 10, "Service Health", "", 1, "L", false, 0, "")
	y += 12

	// Health metrics box
	drawOverviewCard(pdf, 20, y, 257, 30)
	pdf.SetFont("Helvetica", "", 11)

	// Services with downtime
	pdf.SetXY(25, y+7)
	pdf.CellFormat(80, 6, "Services with downtime:", "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", 11)
	if totalDowntimeServices > 0 {
		pdf.SetTextColor(255, 0, 0)
	} else {
		pdf.SetTextColor(0, 150, 0)
	}
	pdf.CellFormat(30, 6, fmt.Sprintf("%d", totalDowntimeServices), "", 0, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)

	// SLA violations
	pdf.SetFont("Helvetica", "", 11)
	pdf.SetXY(145, y+7)
	pdf.CellFormat(80, 6, "SLA violations:", "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", 11)
	if totalSLAViolations > 0 {
		pdf.SetTextColor(255, 0, 0)
	} else {
		pdf.SetTextColor(0, 150, 0)
	}
	pdf.CellFormat(30, 6, fmt.Sprintf("%d", totalSLAViolations), "", 0, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)

	// SLA requirement
	pdf.SetFont("Helvetica", "", 11)
	pdf.SetXY(25, y+17)
	pdf.CellFormat(80, 6, "SLA requirement:", "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(30, 6, fmt.Sprintf("%.2f%%", DefaultSLAPercentage), "", 0, "L", false, 0, "")

	// ===== PAGE 2: MEMBER TABLE =====
	pdf.AddPage()

	y = 40
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetXY(10, y)
	pdf.CellFormat(277, 10, "Member Performance Details", "", 1, "L", false, 0, "")
	y += 12

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

	// Table rows with alternating colors
	fillToggle := false
	for _, row := range memberData {
		if y > 180 {
			pdf.AddPage()
			y = 40

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
			pdf.CellFormat(colStatusW, rowH, "[OK]", "1", 1, "C", true, 0, "")
		} else {
			pdf.SetTextColor(255, 0, 0)
			pdf.CellFormat(colStatusW, rowH, "[FAIL]", "1", 1, "C", true, 0, "")
		}
		pdf.SetTextColor(0, 0, 0)

		y += rowH
	}

	// Total row
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetFillColor(230, 230, 230)
	totalColWidth := colMemberW + colLevelW + colRequestsW + colPercentW + colServicesW + colDowntimeW + colUptimeW
	pdf.SetXY(10, y)
	pdf.CellFormat(totalColWidth, rowH, "TOTALS", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colBaseCostW, rowH, fmt.Sprintf("$%.2f", grandTotalBase), "1", 0, "R", true, 0, "")
	pdf.CellFormat(colBilledW, rowH, fmt.Sprintf("$%.2f", grandTotalBilled), "1", 0, "R", true, 0, "")
	pdf.CellFormat(colStatusW, rowH, "", "1", 1, "C", true, 0, "")

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

func drawOverviewCard(pdf *gofpdf.Fpdf, x, y, w, h float64) {
	pdf.SetDrawColor(200, 200, 200)
	pdf.SetLineWidth(0.3)
	pdf.Rect(x, y, w, h, "D")
	pdf.SetDrawColor(0, 0, 0)
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
