package billing

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	cfg "ibp-geodns/src/common/config"
	data2 "ibp-geodns/src/common/data2"
	log "ibp-geodns/src/common/logging"

	"github.com/phpdave11/gofpdf"
)

// CountryRequestData holds country request data with name
type CountryRequestData struct {
	Hits        int
	CountryName string
}

// CountryStats holds statistics for a country
type CountryStats struct {
	Country       string
	CountryName   string
	Requests      int
	Percentage    float64
	Change1Month  float64
	Change3Months float64
	Change6Months float64
}

// ServiceStats holds statistics for a service
type ServiceStats struct {
	Service       string
	Requests      int
	Percentage    float64
	Change1Month  float64
	Change3Months float64
	Change6Months float64
}

// writeMonthlyOverviewPDF generates a summary PDF for all members with modern design
func writeMonthlyOverviewPDF(sum *Summary, sla SLASummary, outDir string, month time.Time) error {
	logoPath := findLogo(filepath.Dir(outDir))
	filename := filepath.Join(outDir, fmt.Sprintf("%s-Monthly_Overview.pdf", month.Format("2006_01")))

	pdf := gofpdf.New("L", "mm", "A4", "") // Landscape
	pdf.SetTitle("IBP Monthly Billing Sheet", false)
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
		pdf.CellFormat(200, 10, "IBP Monthly Billing Sheet", "", 0, "L", false, 0, "")

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

		// Sort by level descending, then alphabetically
		sort.Slice(memberData, func(i, j int) bool {
			if memberData[i].level != memberData[j].level {
				return memberData[i].level > memberData[j].level
			}
			return memberData[i].name < memberData[j].name
		})

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

	// ===== PAGE 2: SERVICE HEALTH =====
	pdf.AddPage()
	y = 40

	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetXY(20, y)
	pdf.CellFormat(257, 10, "Service Health", "", 1, "L", false, 0, "")
	y += 12

	// Health metrics box
	drawCard(pdf, 20, y, 257, 30)
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

	// Add downtime calendar
	y += 35 // Reduced from 40
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetXY(20, y)
	pdf.CellFormat(257, 8, "Downtime Calendar", "", 1, "L", false, 0, "")
	y += 8 // Reduced from 10

	// Draw calendar with adjusted dimensions
	drawDowntimeCalendar(pdf, 30, y, 237, month) // Reduced width from 257 to 237 (about 8% reduction), moved x from 20 to 30

	// ===== PAGE 3: MEMBER BILLINGS TABLE =====
	pdf.AddPage()

	// Calculate table dimensions
	const tableWidth = 280.0
	pageWidth := 297.0 // A4 landscape width
	tableX := (pageWidth - tableWidth) / 2

	// Title
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetXY(tableX, 40)
	pdf.CellFormat(tableWidth, 10, "Member Billings", "", 1, "L", false, 0, "")

	// Calculate vertical centering
	startY := 48.0

	// Estimate table height: header + (rows * rowH) + total row
	estimatedHeight := 8.0 + float64(len(memberData))*8.0 + 8.0
	availableHeight := 190.0 - startY // From startY to before footer

	if estimatedHeight < availableHeight {
		startY += (availableHeight - estimatedHeight) / 2
	}

	y = startY

	// Table setup (adjust column widths for centering)
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

	// Table header
	pdf.SetFillColor(50, 50, 50)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetXY(tableX, y)
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
			pdf.SetXY(tableX, y)
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

		pdf.SetXY(tableX, y)

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
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetFillColor(230, 230, 230)
	totalColWidth := colMemberW + colLevelW + colRequestsW + colPercentW + colServicesW + colDowntimeW + colUptimeW
	pdf.SetXY(tableX, y)
	pdf.CellFormat(totalColWidth, rowH, "TOTALS", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colBaseCostW, rowH, fmt.Sprintf("$%.2f", grandTotalBase), "1", 0, "R", true, 0, "")
	pdf.CellFormat(colBilledW, rowH, fmt.Sprintf("$%.2f", grandTotalBilled), "1", 0, "R", true, 0, "")
	pdf.CellFormat(colStatusW, rowH, "", "1", 1, "C", true, 0, "")

	// ===== PAGE 4: GEOGRAPHIC DISTRIBUTION =====
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetXY(20, 40)
	pdf.CellFormat(257, 10, "Geographic Distribution - Top 10", "", 1, "L", false, 0, "")

	// Get country statistics
	countryStats := getCountryStatistics(month)

	// Center the table
	const geoTableWidth = 252.0
	geoTableX := (297.0 - geoTableWidth) / 2

	// Draw unified country table
	drawUnifiedCountryTable(pdf, countryStats, 55, geoTableX, geoTableWidth)

	// ===== PAGE 5: SERVICE/CHAIN DISTRIBUTION =====
	pdf.AddPage()
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetXY(20, 40)
	pdf.CellFormat(257, 10, "Service/Chain Distribution - Top 10", "", 1, "L", false, 0, "")

	// Get service statistics
	serviceStats := getServiceStatistics(month)

	// Center the table
	const svcTableWidth = 252.0
	svcTableX := (297.0 - svcTableWidth) / 2

	// Draw unified service table
	drawUnifiedServiceTable(pdf, serviceStats, 55, svcTableX, svcTableWidth)

	if err := pdf.OutputFileAndClose(filename); err != nil {
		return err
	}

	log.Log(log.Info, "[billing] Monthly overview PDF written → %s", filename)
	return nil
}

// drawUnifiedCountryTable draws a single table with all 15 countries
func drawUnifiedCountryTable(pdf *gofpdf.Fpdf, stats []CountryStats, startY, tableX, tableWidth float64) {
	if len(stats) == 0 {
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetXY(10, startY)
		pdf.CellFormat(277, 10, "No country statistics available", "", 1, "C", false, 0, "")
		return
	}

	// Column widths - adjusted for better proportions with 15% increase
	const (
		colRankW     = 17.0
		colCountryW  = 85.0
		colRequestsW = 42.0
		colShareW    = 30.0
		colChange1W  = 26.0
		colChange3W  = 26.0
		colChange6W  = 26.0
		rowH         = 10.0
	)

	// Calculate total width and center the table
	totalWidth := colRankW + colCountryW + colRequestsW + colShareW + colChange1W + colChange3W + colChange6W
	x := (297.0 - totalWidth) / 2
	y := startY

	// Table header with modern style
	pdf.SetFillColor(50, 50, 50)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 11)
	x = tableX
	y = startY

	pdf.SetXY(x, y)
	pdf.CellFormat(colRankW, rowH, "#", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colCountryW, rowH, "Country", "1", 0, "L", true, 0, "")
	pdf.CellFormat(colRequestsW, rowH, "Requests", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colShareW, rowH, "Share", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colChange1W, rowH, "1M Ago", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colChange3W, rowH, "3M Ago", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colChange6W, rowH, "6M Ago", "1", 1, "R", true, 0, "")

	pdf.SetTextColor(0, 0, 0)
	y += rowH

	// Data rows with alternating colors
	pdf.SetFont("Helvetica", "", 10)
	fillToggle := false

	for i := 0; i < 10 && i < len(stats); i++ {
		if y > 180 {
			pdf.AddPage()
			y = 40
			// Reprint header with consistent styling
			pdf.SetFillColor(50, 50, 50)
			pdf.SetTextColor(255, 255, 255)
			pdf.SetFont("Helvetica", "B", 11)
			pdf.SetXY(x, y)
			pdf.CellFormat(colRankW, rowH, "#", "1", 0, "C", true, 0, "")
			pdf.CellFormat(colCountryW, rowH, "Country", "1", 0, "L", true, 0, "")
			pdf.CellFormat(colRequestsW, rowH, "Requests", "1", 0, "R", true, 0, "")
			pdf.CellFormat(colShareW, rowH, "Share", "1", 0, "R", true, 0, "")
			pdf.CellFormat(colChange1W, rowH, "1M Ago", "1", 0, "R", true, 0, "")
			pdf.CellFormat(colChange3W, rowH, "3M Ago", "1", 0, "R", true, 0, "")
			pdf.CellFormat(colChange6W, rowH, "6M Ago", "1", 1, "R", true, 0, "")
			pdf.SetTextColor(0, 0, 0)
			pdf.SetFont("Helvetica", "", 10)
			y += rowH
		}

		fillToggle = !fillToggle
		if fillToggle {
			pdf.SetFillColor(245, 245, 245)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		pdf.SetXY(x, y)
		pdf.CellFormat(colRankW, rowH, fmt.Sprintf("%d", i+1), "1", 0, "C", fillToggle, 0, "")
		pdf.CellFormat(colCountryW, rowH, stats[i].CountryName, "1", 0, "L", fillToggle, 0, "")
		pdf.CellFormat(colRequestsW, rowH, formatNumber(stats[i].Requests), "1", 0, "R", fillToggle, 0, "")
		pdf.CellFormat(colShareW, rowH, fmt.Sprintf("%.1f%%", stats[i].Percentage), "1", 0, "R", fillToggle, 0, "")

		// Change colors
		changeStr1 := formatChange(stats[i].Change1Month)
		if stats[i].Change1Month > 0 {
			pdf.SetTextColor(0, 150, 0)
		} else if stats[i].Change1Month < 0 {
			pdf.SetTextColor(255, 0, 0)
		}
		pdf.CellFormat(colChange1W, rowH, changeStr1, "1", 0, "R", fillToggle, 0, "")
		pdf.SetTextColor(0, 0, 0)

		changeStr3 := formatChange(stats[i].Change3Months)
		if stats[i].Change3Months > 0 {
			pdf.SetTextColor(0, 150, 0)
		} else if stats[i].Change3Months < 0 {
			pdf.SetTextColor(255, 0, 0)
		}
		pdf.CellFormat(colChange3W, rowH, changeStr3, "1", 0, "R", fillToggle, 0, "")
		pdf.SetTextColor(0, 0, 0)

		changeStr6 := formatChange(stats[i].Change6Months)
		if stats[i].Change6Months > 0 {
			pdf.SetTextColor(0, 150, 0)
		} else if stats[i].Change6Months < 0 {
			pdf.SetTextColor(255, 0, 0)
		}
		pdf.CellFormat(colChange6W, rowH, changeStr6, "1", 1, "R", fillToggle, 0, "")
		pdf.SetTextColor(0, 0, 0)

		y += rowH
	}
}

// drawUnifiedServiceTable draws a single table with all 15 services
func drawUnifiedServiceTable(pdf *gofpdf.Fpdf, stats []ServiceStats, startY, tableX, tableWidth float64) {
	if len(stats) == 0 {
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetXY(10, startY)
		pdf.CellFormat(277, 10, "No service statistics available", "", 1, "C", false, 0, "")
		return
	}

	const (
		colRankW     = 17.0
		colServiceW  = 85.0
		colRequestsW = 42.0
		colShareW    = 30.0
		colChange1W  = 26.0
		colChange3W  = 26.0
		colChange6W  = 26.0
		rowH         = 10.0
	)

	// Calculate total width and center the table
	totalWidth := colRankW + colServiceW + colRequestsW + colShareW + colChange1W + colChange3W + colChange6W
	x := (297.0 - totalWidth) / 2
	y := startY

	// Table header
	pdf.SetFillColor(50, 50, 50)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 11)
	x = tableX
	y = startY

	pdf.SetXY(x, y)
	pdf.CellFormat(colRankW, rowH, "#", "1", 0, "C", true, 0, "")
	pdf.CellFormat(colServiceW, rowH, "Service/Chain", "1", 0, "L", true, 0, "")
	pdf.CellFormat(colRequestsW, rowH, "Requests", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colShareW, rowH, "Share", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colChange1W, rowH, "1M Ago", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colChange3W, rowH, "3M Ago", "1", 0, "R", true, 0, "")
	pdf.CellFormat(colChange6W, rowH, "6M Ago", "1", 1, "R", true, 0, "")

	pdf.SetTextColor(0, 0, 0)
	y += rowH

	// Data rows
	pdf.SetFont("Helvetica", "", 10)
	fillToggle := false

	for i := 0; i < 10 && i < len(stats); i++ {
		if y > 180 {
			pdf.AddPage()
			y = 40
			// Reprint header
			pdf.SetFillColor(50, 50, 50)
			pdf.SetTextColor(255, 255, 255)
			pdf.SetFont("Helvetica", "B", 10)
			pdf.SetXY(x, y)
			pdf.CellFormat(colRankW, rowH, "#", "1", 0, "C", true, 0, "")
			pdf.CellFormat(colServiceW, rowH, "Service/Chain", "1", 0, "L", true, 0, "")
			pdf.CellFormat(colRequestsW, rowH, "Requests", "1", 0, "R", true, 0, "")
			pdf.CellFormat(colShareW, rowH, "Share", "1", 0, "R", true, 0, "")
			pdf.CellFormat(colChange1W, rowH, "1M Ago", "1", 0, "R", true, 0, "")
			pdf.CellFormat(colChange3W, rowH, "3M Ago", "1", 0, "R", true, 0, "")
			pdf.CellFormat(colChange6W, rowH, "6M Ago", "1", 1, "R", true, 0, "")
			pdf.SetTextColor(0, 0, 0)
			pdf.SetFont("Helvetica", "", 10)
			y += rowH
		}

		fillToggle = !fillToggle
		if fillToggle {
			pdf.SetFillColor(245, 245, 245)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}

		// Convert domain to service name
		serviceName := domainToServiceName(stats[i].Service)
		if len(serviceName) > 35 {
			serviceName = serviceName[:32] + "..."
		}

		pdf.SetXY(x, y)
		pdf.CellFormat(colRankW, rowH, fmt.Sprintf("%d", i+1), "1", 0, "C", fillToggle, 0, "")
		pdf.CellFormat(colServiceW, rowH, serviceName, "1", 0, "L", fillToggle, 0, "")
		pdf.CellFormat(colRequestsW, rowH, formatNumber(stats[i].Requests), "1", 0, "R", fillToggle, 0, "")
		pdf.CellFormat(colShareW, rowH, fmt.Sprintf("%.1f%%", stats[i].Percentage), "1", 0, "R", fillToggle, 0, "")

		// 1M change
		changeStr1 := formatChange(stats[i].Change1Month)
		if stats[i].Change1Month > 0 {
			pdf.SetTextColor(0, 150, 0)
		} else if stats[i].Change1Month < 0 {
			pdf.SetTextColor(255, 0, 0)
		}
		pdf.CellFormat(colChange1W, rowH, changeStr1, "1", 0, "R", fillToggle, 0, "")
		pdf.SetTextColor(0, 0, 0)

		// 3M change
		changeStr3 := formatChange(stats[i].Change3Months)
		if stats[i].Change3Months > 0 {
			pdf.SetTextColor(0, 150, 0)
		} else if stats[i].Change3Months < 0 {
			pdf.SetTextColor(255, 0, 0)
		}
		pdf.CellFormat(colChange3W, rowH, changeStr3, "1", 0, "R", fillToggle, 0, "")
		pdf.SetTextColor(0, 0, 0)

		// 6M change
		changeStr6 := formatChange(stats[i].Change6Months)
		if stats[i].Change6Months > 0 {
			pdf.SetTextColor(0, 150, 0)
		} else if stats[i].Change6Months < 0 {
			pdf.SetTextColor(255, 0, 0)
		}
		pdf.CellFormat(colChange6W, rowH, changeStr6, "1", 1, "R", fillToggle, 0, "")
		pdf.SetTextColor(0, 0, 0)

		y += rowH
	}
}

// getCountryStatistics retrieves country statistics with historical comparisons
func getCountryStatistics(month time.Time) []CountryStats {
	if data2.DB == nil {
		return []CountryStats{}
	}

	// Get current month stats
	currentStats := getCountryRequestsForMonth(month)

	// Get historical stats
	oneMonthAgo := getCountryRequestsForMonth(month.AddDate(0, -1, 0))
	threeMonthsAgo := getCountryRequestsForMonth(month.AddDate(0, -3, 0))
	sixMonthsAgo := getCountryRequestsForMonth(month.AddDate(0, -6, 0))

	// Calculate total for percentages
	totalRequests := 0
	for _, data := range currentStats {
		totalRequests += data.Hits
	}

	// Build stats with comparisons
	var stats []CountryStats
	for country, data := range currentStats {
		stat := CountryStats{
			Country:     country,
			CountryName: data.CountryName, // Use from database
			Requests:    data.Hits,
			Percentage:  0,
		}

		if totalRequests > 0 {
			stat.Percentage = float64(data.Hits) / float64(totalRequests) * 100.0
		}

		// Calculate changes
		if prev, exists := oneMonthAgo[country]; exists && prev.Hits > 0 {
			stat.Change1Month = ((float64(data.Hits) - float64(prev.Hits)) / float64(prev.Hits)) * 100.0
		}
		if prev, exists := threeMonthsAgo[country]; exists && prev.Hits > 0 {
			stat.Change3Months = ((float64(data.Hits) - float64(prev.Hits)) / float64(prev.Hits)) * 100.0
		}
		if prev, exists := sixMonthsAgo[country]; exists && prev.Hits > 0 {
			stat.Change6Months = ((float64(data.Hits) - float64(prev.Hits)) / float64(prev.Hits)) * 100.0
		}

		stats = append(stats, stat)
	}

	// Sort by requests descending
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Requests > stats[j].Requests
	})

	return stats
}

// getCountryRequestsForMonth gets request counts by country for a specific month
func getCountryRequestsForMonth(month time.Time) map[string]CountryRequestData {
	result := make(map[string]CountryRequestData)

	if data2.DB == nil {
		return result
	}

	startDate := month.Format("2006-01-02")
	endDate := month.AddDate(0, 1, 0).Add(-24 * time.Hour).Format("2006-01-02")

	query := `
        SELECT 
            COALESCE(country_code, 'XX') as country,
            COALESCE(MAX(country_name), 'Unknown') as country_name,
            SUM(hits) as total_hits
        FROM requests
        WHERE date >= ? AND date <= ?
        GROUP BY country_code
        ORDER BY total_hits DESC
    `

	rows, err := data2.DB.Query(query, startDate, endDate)
	if err != nil {
		log.Log(log.Error, "[billing] Failed to query country stats: %v", err)
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var country, countryName string
		var hits int
		if err := rows.Scan(&country, &countryName, &hits); err == nil {
			result[country] = CountryRequestData{
				Hits:        hits,
				CountryName: countryName,
			}
		}
	}

	return result
}

// getServiceStatistics retrieves service statistics with historical comparisons
func getServiceStatistics(month time.Time) []ServiceStats {
	if data2.DB == nil {
		return []ServiceStats{}
	}

	// Get current month stats
	currentStats := getServiceRequestsForMonth(month)

	// Get historical stats
	oneMonthAgo := getServiceRequestsForMonth(month.AddDate(0, -1, 0))
	threeMonthsAgo := getServiceRequestsForMonth(month.AddDate(0, -3, 0))
	sixMonthsAgo := getServiceRequestsForMonth(month.AddDate(0, -6, 0))

	// Calculate total for percentages
	totalRequests := 0
	for _, count := range currentStats {
		totalRequests += count
	}

	// Build stats with comparisons
	var stats []ServiceStats
	for service, requests := range currentStats {
		// Skip the system domains
		if strings.Contains(service, "sys.dotters.network") ||
			strings.Contains(service, "rpc.dotters.network") {
			continue
		}

		stat := ServiceStats{
			Service:    service,
			Requests:   requests,
			Percentage: 0,
		}

		if totalRequests > 0 {
			stat.Percentage = float64(requests) / float64(totalRequests) * 100.0
		}

		// Calculate changes
		if prev, exists := oneMonthAgo[service]; exists && prev > 0 {
			stat.Change1Month = ((float64(requests) - float64(prev)) / float64(prev)) * 100.0
		}
		if prev, exists := threeMonthsAgo[service]; exists && prev > 0 {
			stat.Change3Months = ((float64(requests) - float64(prev)) / float64(prev)) * 100.0
		}
		if prev, exists := sixMonthsAgo[service]; exists && prev > 0 {
			stat.Change6Months = ((float64(requests) - float64(prev)) / float64(prev)) * 100.0
		}

		stats = append(stats, stat)
	}

	// Sort by requests descending
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Requests > stats[j].Requests
	})

	return stats
}

// getServiceRequestsForMonth gets request counts by service domain for a specific month
func getServiceRequestsForMonth(month time.Time) map[string]int {
	result := make(map[string]int)

	if data2.DB == nil {
		return result
	}

	startDate := month.Format("2006-01-02")
	endDate := month.AddDate(0, 1, 0).Add(-24 * time.Hour).Format("2006-01-02")

	query := `
		SELECT 
			domain_name,
			SUM(hits) as total_hits
		FROM requests
		WHERE date >= ? AND date <= ?
		AND domain_name != ''
		GROUP BY domain_name
		ORDER BY total_hits DESC
	`

	rows, err := data2.DB.Query(query, startDate, endDate)
	if err != nil {
		log.Log(log.Error, "[billing] Failed to query service stats: %v", err)
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var domain string
		var hits int
		if err := rows.Scan(&domain, &hits); err == nil {
			result[domain] = hits
		}
	}

	return result
}

// Helper functions remain the same...
func drawGradientCard(pdf *gofpdf.Fpdf, x, y, w, h float64, r, g, b int) {
	// Simple solid color card with shadow effect
	pdf.SetFillColor(r-20, g-20, b-20)
	pdf.Rect(x+1, y+1, w, h, "F")
	pdf.SetFillColor(r, g, b)
	pdf.Rect(x, y, w, h, "F")
}

func drawCard(pdf *gofpdf.Fpdf, x, y, w, h float64) {
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

// formatChange formats a percentage change value
func formatChange(change float64) string {
	if change == 0 {
		return "-"
	}
	sign := "+"
	if change < 0 {
		sign = ""
	}
	return fmt.Sprintf("%s%.1f%%", sign, change)
}

// drawDowntimeCalendar draws a monthly calendar with downtime indicators
func drawDowntimeCalendar(pdf *gofpdf.Fpdf, x, y, width float64, month time.Time) {
	// Get downtime events for the month
	downtimeByDay := getDowntimeByDay(month)

	// Calendar setup
	daysInMonth := time.Date(month.Year(), month.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	firstDay := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	startWeekday := int(firstDay.Weekday())

	// Adjusted dimensions - slightly increased height
	cellWidth := width / 7
	cellHeight := 15.0  // Increased from 13.5
	headerHeight := 6.0 // Increased from 5.0

	// Draw day headers
	days := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	pdf.SetFillColor(50, 50, 50)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 8) // Increased from 7

	for i, day := range days {
		pdf.SetXY(x+float64(i)*cellWidth, y)
		pdf.CellFormat(cellWidth, headerHeight, day, "1", 0, "C", true, 0, "")
	}

	pdf.SetTextColor(0, 0, 0)
	y += headerHeight

	// Draw calendar days
	pdf.SetFont("Helvetica", "", 8) // Increased from 7
	week := 0
	for day := 1; day <= daysInMonth; day++ {
		col := (startWeekday + day - 1) % 7
		if day > 1 && col == 0 {
			week++
		}

		cellX := x + float64(col)*cellWidth
		cellY := y + float64(week)*cellHeight

		// Determine cell color based on downtime
		downtime := downtimeByDay[day]
		if downtime > 0 {
			if downtime >= 5 {
				pdf.SetFillColor(255, 200, 200)
			} else if downtime >= 3 {
				pdf.SetFillColor(255, 230, 200)
			} else {
				pdf.SetFillColor(255, 255, 200)
			}
		} else {
			pdf.SetFillColor(200, 255, 200)
		}

		// Draw cell
		pdf.Rect(cellX, cellY, cellWidth, cellHeight, "FD")

		// Add day number
		pdf.SetXY(cellX, cellY+1)
		pdf.CellFormat(cellWidth, 5, fmt.Sprintf("%d", day), "", 0, "C", false, 0, "")

		// Add downtime count if any
		if downtime > 0 {
			pdf.SetFont("Helvetica", "", 6) // Increased from 5
			pdf.SetTextColor(100, 100, 100)
			pdf.SetXY(cellX, cellY+7) // Adjusted from 6
			pdf.CellFormat(cellWidth, 4, fmt.Sprintf("%d", downtime), "", 0, "C", false, 0, "")
			pdf.SetTextColor(0, 0, 0)
			pdf.SetFont("Helvetica", "", 8)
		}
	}

	// Legend (slightly larger)
	legendY := y + float64(week+1)*cellHeight + 5
	pdf.SetFont("Helvetica", "", 7) // Increased from 6
	pdf.SetXY(x, legendY)
	pdf.CellFormat(25, 4, "Legend:", "", 0, "L", false, 0, "")

	legendBoxSize := 12.0 // Increased from 10.0
	legendHeight := 4.0   // Increased from 3.0

	// Green
	pdf.SetFillColor(200, 255, 200)
	pdf.Rect(x+30, legendY, legendBoxSize, legendHeight, "FD")
	pdf.SetXY(x+43, legendY)
	pdf.CellFormat(22, legendHeight, "No issues", "", 0, "L", false, 0, "")

	// Yellow
	pdf.SetFillColor(255, 255, 200)
	pdf.Rect(x+70, legendY, legendBoxSize, legendHeight, "FD")
	pdf.SetXY(x+83, legendY)
	pdf.CellFormat(22, legendHeight, "1-2 events", "", 0, "L", false, 0, "")

	// Orange
	pdf.SetFillColor(255, 230, 200)
	pdf.Rect(x+110, legendY, legendBoxSize, legendHeight, "FD")
	pdf.SetXY(x+123, legendY)
	pdf.CellFormat(22, legendHeight, "3-4 events", "", 0, "L", false, 0, "")

	// Red
	pdf.SetFillColor(255, 200, 200)
	pdf.Rect(x+150, legendY, legendBoxSize, legendHeight, "FD")
	pdf.SetXY(x+163, legendY)
	pdf.CellFormat(22, legendHeight, "5+ events", "", 0, "L", false, 0, "")
}

// getDowntimeByDay returns a map of day -> downtime event count
func getDowntimeByDay(month time.Time) map[int]int {
	result := make(map[int]int)

	if data2.DB == nil {
		return result
	}

	startTime := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	endTime := startTime.AddDate(0, 1, 0).Add(-time.Second)

	query := `
        SELECT 
            DAY(start_time) as day,
            COUNT(*) as event_count
        FROM member_events
        WHERE status = 0
        AND start_time >= ? AND start_time < ?
        GROUP BY DAY(start_time)
    `

	rows, err := data2.DB.Query(query, startTime, endTime)
	if err != nil {
		log.Log(log.Error, "[billing] Failed to query downtime by day: %v", err)
		return result
	}
	defer rows.Close()

	for rows.Next() {
		var day, count int
		if err := rows.Scan(&day, &count); err == nil {
			result[day] = count
		}
	}

	return result
}

// domainToServiceName converts a domain like "hydration.dotters.network" to "HYDRATION"
func domainToServiceName(domain string) string {
	// Map domains to service names based on configuration
	c := cfg.GetConfig()
	for serviceName, service := range c.Services {
		for _, provider := range service.Providers {
			for _, rpcUrl := range provider.RpcUrls {
				if strings.Contains(strings.ToLower(rpcUrl), strings.ToLower(domain)) {
					return serviceName
				}
			}
		}
	}

	// Fallback: clean up the domain name
	name := strings.TrimSuffix(domain, ".dotters.network")
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, ".", " ")

	// Convert to uppercase
	return strings.ToUpper(name)
}
