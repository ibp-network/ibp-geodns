package billing

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	cfg "ibp-geodns/src/common/config"
	data2 "ibp-geodns/src/common/data2"
	log "ibp-geodns/src/common/logging"

	"github.com/phpdave11/gofpdf"
)

/*
	---------------------------------------------------------------------
	                            watermark helpers

---------------------------------------------------------------------
*/
func findLogo(baseDir string) string {
	// Try multiple possible locations for the logo
	possiblePaths := []string{
		filepath.Join(baseDir, "public", "static", "imgs", "ibp.png"),
		filepath.Join(baseDir, "ibp.png"),
		filepath.Join(baseDir, "..", "assets", "ibp.png"),
		filepath.Join(baseDir, "..", "ibp.png"),
		"/opt/ibp-geodns/assets/ibp.png",
	}

	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			log.Log(log.Debug, "[billing] Found logo at: %s", p)
			return p
		}
	}

	log.Log(log.Warn, "[billing] Logo not found in any of the expected locations")
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

/*
	---------------------------------------------------------------------
	                     "cost by service" — PDF report

---------------------------------------------------------------------
*/
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

/*
	---------------------------------------------------------------------
	                 "billing by member" — PDF report

---------------------------------------------------------------------
*/
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

// MemberStats holds DNS request statistics for a member
type MemberStats struct {
	RequestCount int
}

// calculateMemberStats queries the database for member request statistics
// Updated to use member's Details.Name for database lookup
func calculateMemberStats(month time.Time) map[string]MemberStats {
	stats := make(map[string]MemberStats)

	// Check if database is initialized
	if data2.DB == nil {
		log.Log(log.Error, "[billing] Database not initialized for member stats calculation")
		return stats
	}

	// Get configuration to map member IDs to their Details.Name
	c := cfg.GetConfig()
	nameToMemberID := make(map[string]string)

	// Build reverse mapping from Details.Name to member ID
	for memberID, member := range c.Members {
		if member.Details.Name != "" {
			nameToMemberID[member.Details.Name] = memberID
		} else {
			// Fallback to member ID if Details.Name is empty
			nameToMemberID[memberID] = memberID
		}
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
			// Store stats using the Details.Name as key
			stats[memberName] = MemberStats{
				RequestCount: totalHits,
			}

			// Also store using member ID if we have a mapping
			if memberID, exists := nameToMemberID[memberName]; exists && memberID != memberName {
				stats[memberID] = MemberStats{
					RequestCount: totalHits,
				}
			}
		}
	}

	return stats
}

/* --------------------------------------------------------------------- */

func Version() string {
	return "v0.4.8"
}
