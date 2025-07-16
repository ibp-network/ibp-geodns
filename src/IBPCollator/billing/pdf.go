package billing

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"time"

	cfg "ibp-geodns/src/common/config"
	data2 "ibp-geodns/src/common/data2"
	log "ibp-geodns/src/common/logging"

	"github.com/phpdave11/gofpdf"
)

/* ---------------------------------------------------------------------
                            watermark helpers
--------------------------------------------------------------------- */

func findLogo(baseDir string) string {
	p := filepath.Join(baseDir, "assets", "ibp.png")
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

// addPageWithWatermark creates a new page, draws the centred logo (62.5%
// width, 25% transparency) and **moves Y to 32mm** so subsequent content
// never collides with the header/title.
func addPageWithWatermark(pdf *gofpdf.Fpdf, logo string) {
	pdf.AddPage()
	if logo != "" {
		pageW, pageH := pdf.GetPageSize()
		imgW := pageW * 0.625 // 62.5%
		info := pdf.RegisterImageOptions(logo,
			gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true})
		nativeW, nativeH := info.Extent()
		scale := imgW / nativeW
		imgH := nativeH * scale
		imgX := (pageW - imgW) / 2
		imgY := (pageH - imgH) / 2
		pdf.SetAlpha(0.25, "Normal")
		pdf.ImageOptions(logo, imgX, imgY, imgW, 0,
			false, gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, 0, "")
		pdf.SetAlpha(1, "Normal")
	}
	// Reserve vertical space so data never overlaps the header
	pdf.SetY(32.0)
}

/* ---------------------------------------------------------------------
                         reflection convenience
--------------------------------------------------------------------- */

func lookupString(obj interface{}, field string) (string, bool) {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return "", false
	}
	f := v.FieldByName(field)
	if f.IsValid() && f.Kind() == reflect.String {
		return f.String(), true
	}
	return "", false
}

func lookupInt64(obj interface{}, field string) (int64, bool) {
	v := reflect.ValueOf(obj)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return 0, false
	}
	f := v.FieldByName(field)
	if f.IsValid() {
		switch f.Kind() {
		case reflect.Int, reflect.Int32, reflect.Int64:
			return f.Int(), true
		case reflect.Uint, reflect.Uint32, reflect.Uint64:
			return int64(f.Uint()), true
		}
	}
	return 0, false
}

/* ---------------------------------------------------------------------
                     "cost by service" — PDF report
--------------------------------------------------------------------- */

func writeServiceCostPDF(sum *Summary, tmpDir string) error {
	c := cfg.GetConfig()
	logoPath := findLogo(c.Local.System.WorkDir)

	const title = "IBP Network - Cost by Service"
	pdf := gofpdf.New("P", "mm", "A4", "")
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

	// deterministic ordering
	serviceNames := make([]string, 0, len(sum.Services))
	for s := range sum.Services {
		serviceNames = append(serviceNames, s)
	}
	sort.Strings(serviceNames)

	// geometry
	pageW, _ := pdf.GetPageSize()
	const (
		leftIndent = 15.0
		boxGap     = 6.0
		rowH       = 6.0
		colSvcW    = 100.0
		colCostW   = 60.0
	)
	boxWidth := colSvcW + colCostW
	leftMargin := (pageW - boxWidth) / 2
	origLeft, _, _, _ := pdf.GetMargins()

	for _, svc := range serviceNames {
		startY := pdf.GetY()
		if startY > 230 {
			addPageWithWatermark(pdf, logoPath)
			startY = pdf.GetY()
		}

		pdf.SetLeftMargin(leftMargin)
		pdf.SetX(leftMargin)

		// box title = service name
		pdf.SetFont("Helvetica", "B", 12)
		pdf.CellFormat(boxWidth, rowH+2, svc, "", 1, "L", false, 0, "")
		pdf.Ln(1)

		// table header
		pdf.SetFont("Helvetica", "B", 11)
		pdf.SetFillColor(240, 240, 240)
		pdf.CellFormat(colSvcW, rowH, "Member", "1", 0, "L", true, 0, "")
		pdf.CellFormat(colCostW, rowH, "Cost (USD)", "1", 1, "R", true, 0, "")

		pdf.SetFont("Helvetica", "", 10)

		// member list
		sc := sum.Services[svc]
		memberNames := make([]string, 0, len(sc.MemberCosts))
		for m := range sc.MemberCosts {
			memberNames = append(memberNames, m)
		}
		sort.Strings(memberNames)

		fillToggle := false
		for _, mem := range memberNames {
			if pdf.GetY() > 260 {
				// close rectangle, new page
				endY := pdf.GetY()
				pdf.Rect(leftMargin, startY-1, boxWidth, endY-startY+1, "D")
				addPageWithWatermark(pdf, logoPath)
				startY = pdf.GetY()
				pdf.SetLeftMargin(leftMargin)
				pdf.SetX(leftMargin)

				// continuation header
				pdf.SetFont("Helvetica", "B", 12)
				pdf.CellFormat(boxWidth, rowH+2, svc+" (cont'd)", "",
					1, "L", false, 0, "")
				pdf.Ln(1)

				pdf.SetFont("Helvetica", "B", 11)
				pdf.CellFormat(colSvcW, rowH, "Member", "1", 0, "L", true, 0, "")
				pdf.CellFormat(colCostW, rowH, "Cost (USD)", "1", 1, "R", true, 0, "")

				pdf.SetFont("Helvetica", "", 10)
			}

			fillToggle = !fillToggle
			pdf.CellFormat(colSvcW, rowH, mem, "1", 0, "L", fillToggle, 0, "")
			pdf.CellFormat(colCostW, rowH,
				fmt.Sprintf("$%.2f", sc.MemberCosts[mem]),
				"1", 1, "R", fillToggle, 0, "")
		}

		// subtotal
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(colSvcW, rowH, "Service Total", "1", 0, "R", false, 0, "")
		pdf.CellFormat(colCostW, rowH, fmt.Sprintf("$%.2f", sc.Total),
			"1", 1, "R", false, 0, "")

		pdf.SetFont("Helvetica", "", 10)

		// border
		endY := pdf.GetY()
		pdf.Rect(leftMargin, startY-1, boxWidth, endY-startY+1, "D")

		pdf.Ln(boxGap)
		pdf.SetLeftMargin(origLeft)
	}

	// grand total
	grand := 0.0
	for _, sc := range sum.Services {
		grand += sc.Total
	}

	if pdf.GetY() > 260 {
		addPageWithWatermark(pdf, logoPath)
	}

	pdf.SetLeftMargin(leftIndent)
	pdf.SetX(leftIndent)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(colSvcW, rowH+1, "Grand Total", "1", 0, "R", false, 0, "")
	pdf.CellFormat(colCostW, rowH+1, fmt.Sprintf("$%.2f", grand),
		"1", 1, "R", false, 0, "")

	filename := filepath.Join(tmpDir,
		fmt.Sprintf("service_cost_%s.pdf", time.Now().UTC().Format("20060102")))
	if err := pdf.OutputFileAndClose(filename); err != nil {
		return err
	}

	log.Log(log.Info, "[billing] service-cost PDF written → %s", filename)
	return nil
}

/* ---------------------------------------------------------------------
                 "billing by member" — PDF report
--------------------------------------------------------------------- */

func getSLABreakdown(sla SLASummary, member, service string) SLABreakdown {
	if upm, ok := sla[member]; ok {
		if bd, ok2 := upm[service]; ok2 {
			return bd
		}
	}
	// Return default if not found
	return SLABreakdown{
		HoursTotal:   730, // Default month hours
		HoursDown:    0,
		HoursUp:      730,
		Uptime:       100.0,
		SLAThreshold: DefaultSLAPercentage,
		SLAHours:     730 * (DefaultSLAPercentage / 100.0),
		MeetsSLA:     true,
	}
}

func writeMemberBillingPDF(sum *Summary, sla SLASummary, tmpDir string, month time.Time) error {
	baseDir := filepath.Dir(tmpDir)
	logoPath := findLogo(baseDir)

	const titleFmt = "IBP Network - Member Billing (%s %d)"
	title := fmt.Sprintf(titleFmt, month.Format("January"), month.Year())

	pdf := gofpdf.New("P", "mm", "A4", "")
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

	// deterministic order
	memberNames := make([]string, 0, len(sum.Members))
	for m := range sum.Members {
		memberNames = append(memberNames, m)
	}
	sort.Strings(memberNames)

	// geometry
	pageW, _ := pdf.GetPageSize()
	const (
		boxGap      = 8.0
		rowH        = 6.0
		colServiceW = 60.0
		colBaseW    = 30.0
		colUptimeW  = 30.0
		colHoursW   = 30.0
		colBillW    = 30.0
	)
	boxWidth := colServiceW + colBaseW + colUptimeW + colHoursW + colBillW
	leftMargin := (pageW - boxWidth) / 2
	origLeft, _, _, _ := pdf.GetMargins()

	grandTotal := 0.0

	// Get config for resource info
	c := cfg.GetConfig()

	// Calculate total requests and member statistics
	memberStats := calculateMemberStats(month)
	totalRequests := 0
	for _, stats := range memberStats {
		totalRequests += stats.RequestCount
	}

	for _, mem := range memberNames {
		startY := pdf.GetY()
		if startY > 210 {
			addPageWithWatermark(pdf, logoPath)
			startY = pdf.GetY()
		}

		pdf.SetLeftMargin(leftMargin)
		pdf.SetX(leftMargin)

		// Member title
		pdf.SetFont("Helvetica", "B", 12)
		pdf.CellFormat(boxWidth, rowH+3, mem, "", 1, "L", false, 0, "")

		// Calculate member-specific stats
		stats := memberStats[mem]

		// Calculate member total for this billing
		memberTotal := 0.0
		downtimeServices := 0
		for svcName := range sum.Members[mem].ServiceCosts {
			breakdown := getSLABreakdown(sla, mem, svcName)
			if breakdown.HoursDown > 0 {
				downtimeServices++
			}
			baseCost := sum.Members[mem].ServiceCosts[svcName]
			billed := baseCost * (breakdown.Uptime / 100.0)
			memberTotal += billed
		}

		// metadata lines
		pdf.SetFont("Helvetica", "", 9)
		metaLines := make([]string, 0, 10)

		// Get member config
		memberConfig, hasMemberConfig := c.Members[mem]
		if hasMemberConfig {
			if memberConfig.Details.Website != "" {
				metaLines = append(metaLines, "Website: "+memberConfig.Details.Website)
			}
		}

		metaLines = append(metaLines,
			"Billing period: "+month.Format("January 2006"))

		if hasMemberConfig {
			metaLines = append(metaLines,
				fmt.Sprintf("IBP member level: %d", memberConfig.Membership.Level))
		}

		// Add new statistics
		metaLines = append(metaLines,
			fmt.Sprintf("DNS requests served: %d", stats.RequestCount))

		if totalRequests > 0 {
			percentage := float64(stats.RequestCount) / float64(totalRequests) * 100.0
			metaLines = append(metaLines,
				fmt.Sprintf("Percentage of total requests: %.2f%%", percentage))
		}

		metaLines = append(metaLines,
			fmt.Sprintf("Services with downtime: %d", downtimeServices))

		metaLines = append(metaLines,
			fmt.Sprintf("Total amount owed: $%.2f", memberTotal))

		for _, ln := range metaLines {
			pdf.CellFormat(boxWidth, rowH, ln, "", 1, "L", false, 0, "")
		}
		pdf.Ln(1)

		// table header
		pdf.SetFont("Helvetica", "B", 11)
		pdf.SetFillColor(240, 240, 240)
		pdf.CellFormat(colServiceW, rowH, "Service", "1", 0, "L", true, 0, "")
		pdf.CellFormat(colBaseW, rowH, "Base (USD)", "1", 0, "R", true, 0, "")
		pdf.CellFormat(colUptimeW, rowH, "Uptime %", "1", 0, "R", true, 0, "")
		pdf.CellFormat(colHoursW, rowH, "Hours Up", "1", 0, "R", true, 0, "")
		pdf.CellFormat(colBillW, rowH, "Billed (USD)", "1", 1, "R", true, 0, "")

		pdf.SetFont("Helvetica", "", 10)

		memberTotal = 0.0 // Reset for table calculation
		svcNames := make([]string, 0, len(sum.Members[mem].ServiceCosts))
		for s := range sum.Members[mem].ServiceCosts {
			svcNames = append(svcNames, s)
		}
		sort.Strings(svcNames)

		fillToggle := false
		for _, svc := range svcNames {
			if pdf.GetY() > 260 {
				// close rectangle, new page
				endY := pdf.GetY()
				pdf.Rect(leftMargin, startY-1, boxWidth, endY-startY+1, "D")
				addPageWithWatermark(pdf, logoPath)
				startY = pdf.GetY()
				pdf.SetLeftMargin(leftMargin)
				pdf.SetX(leftMargin)

				pdf.SetFont("Helvetica", "B", 12)
				pdf.CellFormat(boxWidth, rowH+3, mem+" (cont'd)", "",
					1, "L", false, 0, "")
				pdf.Ln(1)

				// re-header
				pdf.SetFont("Helvetica", "B", 11)
				pdf.CellFormat(colServiceW, rowH, "Service", "1", 0, "L", true, 0, "")
				pdf.CellFormat(colBaseW, rowH, "Base (USD)", "1", 0, "R", true, 0, "")
				pdf.CellFormat(colUptimeW, rowH, "Uptime %", "1", 0, "R", true, 0, "")
				pdf.CellFormat(colHoursW, rowH, "Hours Up", "1", 0, "R", true, 0, "")
				pdf.CellFormat(colBillW, rowH, "Billed (USD)", "1", 1, "R", true, 0, "")

				pdf.SetFont("Helvetica", "", 10)
			}

			baseCost := sum.Members[mem].ServiceCosts[svc]
			breakdown := getSLABreakdown(sla, mem, svc)
			billed := baseCost * (breakdown.Uptime / 100.0)

			fillToggle = !fillToggle

			// Service name row
			pdf.CellFormat(colServiceW, rowH, svc, "1", 0, "L", fillToggle, 0, "")
			pdf.CellFormat(colBaseW, rowH, fmt.Sprintf("$%.2f", baseCost),
				"1", 0, "R", fillToggle, 0, "")

			// Color code the uptime based on SLA
			if !breakdown.MeetsSLA {
				pdf.SetTextColor(255, 0, 0) // Red for not meeting SLA
			}
			pdf.CellFormat(colUptimeW, rowH, fmt.Sprintf("%.2f", breakdown.Uptime),
				"1", 0, "R", fillToggle, 0, "")
			pdf.SetTextColor(0, 0, 0) // Reset to black

			pdf.CellFormat(colHoursW, rowH, fmt.Sprintf("%.2f", breakdown.HoursUp),
				"1", 0, "R", fillToggle, 0, "")
			pdf.CellFormat(colBillW, rowH, fmt.Sprintf("$%.2f", billed),
				"1", 1, "R", fillToggle, 0, "")

			// Resource details row
			if svcConfig, exists := c.Services[svc]; exists {
				pdf.SetFont("Helvetica", "I", 8)
				pdf.SetTextColor(100, 100, 100)
				resourceText := fmt.Sprintf("   Resources: %d nodes, %.1f cores, %.1f GB RAM, %.1f GB disk, %.1f TB bandwidth",
					svcConfig.Resources.Nodes,
					svcConfig.Resources.Cores,
					svcConfig.Resources.Memory,
					svcConfig.Resources.Disk,
					svcConfig.Resources.Bandwidth)
				pdf.CellFormat(boxWidth, rowH-1, resourceText, "LR", 1, "L", false, 0, "")
				pdf.SetTextColor(0, 0, 0)
				pdf.SetFont("Helvetica", "", 10)
			}

			// SLA info row if not meeting SLA
			if !breakdown.MeetsSLA {
				pdf.SetFont("Helvetica", "I", 8)
				pdf.SetTextColor(255, 0, 0)
				slaText := fmt.Sprintf("   BELOW SLA: Required %.2f%% (%.2f hrs), Actual %.2f hrs down",
					breakdown.SLAThreshold, breakdown.SLAHours, breakdown.HoursDown)
				pdf.CellFormat(boxWidth, rowH-1, slaText, "LR", 1, "L", false, 0, "")
				pdf.SetTextColor(0, 0, 0)
				pdf.SetFont("Helvetica", "", 10)
			}

			memberTotal += billed
			grandTotal += billed
		}

		// subtotal
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(colServiceW+colBaseW+colUptimeW+colHoursW, rowH, "Member Total",
			"1", 0, "R", false, 0, "")
		pdf.CellFormat(colBillW, rowH, fmt.Sprintf("$%.2f", memberTotal),
			"1", 1, "R", false, 0, "")

		pdf.SetFont("Helvetica", "", 10)

		// border
		endY := pdf.GetY()
		pdf.Rect(leftMargin, startY-1, boxWidth, endY-startY+1, "D")

		pdf.Ln(boxGap)
		pdf.SetLeftMargin(origLeft)
	}

	// grand total
	if pdf.GetY() > 260 {
		addPageWithWatermark(pdf, logoPath)
	}

	pdf.SetLeftMargin(leftMargin)
	pdf.SetX(leftMargin)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(colServiceW+colBaseW+colUptimeW+colHoursW, rowH+1, "Grand Total",
		"1", 0, "R", false, 0, "")
	pdf.CellFormat(colBillW, rowH+1, fmt.Sprintf("$%.2f", grandTotal),
		"1", 1, "R", false, 0, "")

	pdf.SetLeftMargin(origLeft)

	filename := filepath.Join(tmpDir,
		fmt.Sprintf("member_billing_%s.pdf", month.Format("200601")))
	if err := pdf.OutputFileAndClose(filename); err != nil {
		return err
	}

	log.Log(log.Info, "[billing] member-billing PDF written → %s", filename)
	return nil
}

// MemberStats holds DNS request statistics for a member
type MemberStats struct {
	RequestCount int
}

// calculateMemberStats queries the database for member request statistics
func calculateMemberStats(month time.Time) map[string]MemberStats {
	stats := make(map[string]MemberStats)

	// Check if database is initialized
	if data2.DB == nil {
		log.Log(log.Error, "[billing] Database not initialized for member stats calculation")
		return stats
	}

	// Calculate the time range for the month
	startDate := month.Format("2006-01-02")
	endDate := month.AddDate(0, 1, 0).Add(-24 * time.Hour).Format("2006-01-02")

	// Query for member request counts
	query := `
		SELECT 
			COALESCE(member_name, '(none)') as member_name,
			SUM(hits) as total_hits
		FROM requests
		WHERE date >= ? AND date <= ?
		GROUP BY member_name
	`

	rows, err := data2.DB.Query(query, startDate, endDate)
	if err != nil {
		log.Log(log.Error, "[billing] Failed to query member stats: %v", err)
		return stats
	}
	defer rows.Close()

	for rows.Next() {
		var memberName string
		var totalHits int

		err := rows.Scan(&memberName, &totalHits)
		if err != nil {
			log.Log(log.Error, "[billing] Failed to scan member stats row: %v", err)
			continue
		}

		if memberName != "(none)" {
			stats[memberName] = MemberStats{
				RequestCount: totalHits,
			}
		}
	}

	return stats
}

/* --------------------------------------------------------------------- */

func Version() string {
	return "v0.4.8"
}
