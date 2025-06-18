package billing

// ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
// ┃  Stake Plus Inc. – IBPCollator Billing PDF helpers  (v0.4.4)      ┃
// ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
//
// Changes in this revision
// ------------------------
// • Watermark
//   – centred on both axes
//   – rendered at 50 % of page‑width
//   – opacity increased from 5 % ➜ 10 % for clearer, but still subtle,
//     visibility
// • Removed all non‑ASCII punctuation (en‑dash, bullet, etc.) that was
//   printing as “funky” glyphs on some PDF viewers / printers.
// • Member‑billing layout
//   – each member now enclosed in its own bordered “box” with 4 mm
//     spacing between boxes for better visual separation
//   – content left‑aligned inside the box; numeric columns right‑aligned
// • Service‑cost report header / column labels updated to ASCII only.
// • Helper functions kept fully backward‑compatible – no impact on the
//   rest of the billing package.

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"

	"github.com/phpdave11/gofpdf"
)

/* ---------------------------------------------------------------------
                            watermark helpers
--------------------------------------------------------------------- */

func findLogo(baseDir string) string {
	candidates := []string{
		filepath.Join(baseDir, "assets", "ibp.png"),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// addPageWithWatermark adds a new page and draws the company logo as a
// faint centred watermark (50 % page‑width, 10 % opacity).
func addPageWithWatermark(pdf *gofpdf.Fpdf, logo string) {
	pdf.AddPage()
	if logo == "" {
		return
	}
	pageW, pageH := pdf.GetPageSize()

	// Desired width = 50 % of page width
	imgW := pageW * 0.5

	// Register the image to obtain its native size (only once per doc)
	info := pdf.RegisterImageOptions(logo, gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true})
	nativeW, nativeH := info.Extent()

	scale := imgW / nativeW
	imgH := nativeH * scale

	imgX := (pageW - imgW) / 2
	imgY := (pageH - imgH) / 2

	pdf.SetAlpha(0.10, "Normal") // 10 % opacity
	pdf.ImageOptions(logo, imgX, imgY, imgW, 0,
		false, gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, 0, "")
	pdf.SetAlpha(1, "Normal")
}

/* ---------------------------------------------------------------------
                     “cost by service”  –  PDF report
--------------------------------------------------------------------- */

func writeServiceCostPDF(sum *Summary, tmpDir string) error {
	c := cfg.GetConfig()
	logoPath := findLogo(c.Local.System.WorkDir)

	pdf := gofpdf.New("P", "mm", "A4", "")
	title := "IBP Network - Cost by Service"
	pdf.SetTitle(title, false)
	pdf.SetAuthor("IBPCollator "+Version(), false)

	pdf.SetHeaderFuncMode(func() {
		pdf.SetFont("Helvetica", "B", 15)
		pdf.CellFormat(0, 10, title, "", 1, "C", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.CellFormat(0, 6, time.Now().UTC().Format("02 Jan 2006 15:04 UTC"),
			"", 0, "C", false, 0, "")
		pdf.Ln(8)
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

	// column layout
	const (
		colServiceW = 60.0
		colMemberW  = 70.0
		colCostW    = 40.0
		rowH        = 6.0
	)

	drawHeader := func() {
		pdf.SetFont("Helvetica", "B", 11)
		pdf.SetFillColor(230, 230, 230)
		pdf.CellFormat(colServiceW, rowH, "Service", "1", 0, "L", true, 0, "")
		pdf.CellFormat(colMemberW, rowH, "Member", "1", 0, "L", true, 0, "")
		pdf.CellFormat(colCostW, rowH, "Cost (USD)", "1", 1, "R", true, 0, "")
	}

	drawHeader()
	pdf.SetFont("Helvetica", "", 10)

	for _, svc := range serviceNames {
		sc := sum.Services[svc]

		// list members for this service
		memberNames := make([]string, 0, len(sc.MemberCosts))
		for m := range sc.MemberCosts {
			memberNames = append(memberNames, m)
		}
		sort.Strings(memberNames)

		for i, mem := range memberNames {
			if pdf.GetY() > 265 {
				addPageWithWatermark(pdf, logoPath)
				drawHeader()
			}

			// Alternate row shading
			fill := i%2 == 1
			if i == 0 {
				pdf.CellFormat(colServiceW, rowH, svc, "1", 0, "L", fill, 0, "")
			} else {
				pdf.CellFormat(colServiceW, rowH, "", "1", 0, "L", fill, 0, "")
			}
			pdf.CellFormat(colMemberW, rowH, mem, "1", 0, "L", fill, 0, "")
			pdf.CellFormat(colCostW, rowH, fmt.Sprintf("$%.2f", sc.MemberCosts[mem]),
				"1", 1, "R", fill, 0, "")
		}

		// service subtotal
		if pdf.GetY() > 265 {
			addPageWithWatermark(pdf, logoPath)
			drawHeader()
		}
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(colServiceW+colMemberW, rowH, "Subtotal",
			"1", 0, "R", false, 0, "")
		pdf.CellFormat(colCostW, rowH, fmt.Sprintf("$%.2f", sc.Total),
			"1", 1, "R", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
	}

	// grand total
	var grand float64
	for _, sc := range sum.Services {
		grand += sc.Total
	}
	if pdf.GetY() > 265 {
		addPageWithWatermark(pdf, logoPath)
		drawHeader()
	}
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(colServiceW+colMemberW, rowH+1, "Grand Total",
		"1", 0, "R", false, 0, "")
	pdf.CellFormat(colCostW, rowH+1, fmt.Sprintf("$%.2f", grand),
		"1", 1, "R", false, 0, "")

	filename := filepath.Join(tmpDir,
		fmt.Sprintf("service_cost_%s.pdf", time.Now().UTC().Format("20060102")))
	if err := pdf.OutputFileAndClose(filename); err != nil {
		return err
	}
	log.Log(log.Info, "[billing] service‑cost PDF written → %s", filename)
	return nil
}

/* ---------------------------------------------------------------------
                 “billing by member”  –  PDF report
--------------------------------------------------------------------- */

func getUptimePercent(sla SLASummary, member, service string) float64 {
	if upm, ok := sla[member]; ok {
		if bd, ok2 := upm[service]; ok2 {
			return bd.Uptime
		}
	}
	return 100.0
}

func writeMemberBillingPDF(sum *Summary, sla SLASummary, tmpDir string, month time.Time) error {
	baseDir := filepath.Dir(tmpDir)
	logoPath := findLogo(baseDir)

	title := fmt.Sprintf("IBP Network - Member Billing (%s %d)",
		month.Format("January"), month.Year())

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(title, false)
	pdf.SetAuthor("IBPCollator "+Version(), false)

	pdf.SetHeaderFuncMode(func() {
		pdf.SetFont("Helvetica", "B", 15)
		pdf.CellFormat(0, 10, title, "", 1, "C", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.CellFormat(0, 6, time.Now().UTC().Format("02 Jan 2006 15:04 UTC"),
			"", 0, "C", false, 0, "")
		pdf.Ln(8)
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

	// column layout
	const (
		leftMargin  = 15.0
		boxWidth    = 180.0 // page width - 2*leftMargin (A4 210mm)
		colServiceW = 60.0
		colBaseW    = 30.0
		colUptimeW  = 30.0
		colBillW    = 40.0
		rowH        = 6.0
		boxGap      = 4.0
	)

	drawTableHeader := func() {
		pdf.SetFont("Helvetica", "B", 11)
		pdf.SetFillColor(230, 230, 230)
		pdf.CellFormat(colServiceW, rowH, "Service", "1", 0, "L", true, 0, "")
		pdf.CellFormat(colBaseW, rowH, "Base (USD)", "1", 0, "R", true, 0, "")
		pdf.CellFormat(colUptimeW, rowH, "Uptime %", "1", 0, "R", true, 0, "")
		pdf.CellFormat(colBillW, rowH, "Billed (USD)", "1", 1, "R", true, 0, "")
	}

	pdf.SetFont("Helvetica", "", 10)
	grandTotal := 0.0

	for _, mem := range memberNames {
		startY := pdf.GetY()

		// Page break – ensure the entire box fits or starts on new page
		if startY > 230 { // leave room for at least ~40mm content
			addPageWithWatermark(pdf, logoPath)
			startY = pdf.GetY()
		}

		pdf.SetX(leftMargin)
		pdf.SetFont("Helvetica", "B", 12)
		pdf.CellFormat(boxWidth, rowH+2, mem, "",
			1, "L", false, 0, "")
		pdf.Ln(1)
		drawTableHeader()
		pdf.SetFont("Helvetica", "", 10)

		memberTotal := 0.0
		svcNames := make([]string, 0, len(sum.Members[mem].ServiceCosts))
		for s := range sum.Members[mem].ServiceCosts {
			svcNames = append(svcNames, s)
		}
		sort.Strings(svcNames)

		for _, svc := range svcNames {
			if pdf.GetY() > 260 {
				// before page break, close rectangle for current box
				endY := pdf.GetY()
				pdf.Rect(leftMargin, startY-1, boxWidth, endY-startY+1, "D")
				addPageWithWatermark(pdf, logoPath)
				startY = pdf.GetY()
				pdf.SetX(leftMargin)
				pdf.SetFont("Helvetica", "B", 12)
				pdf.CellFormat(boxWidth, rowH+2, mem+" (cont'd)", "",
					1, "L", false, 0, "")
				pdf.Ln(1)
				drawTableHeader()
				pdf.SetFont("Helvetica", "", 10)
			}

			baseCost := sum.Members[mem].ServiceCosts[svc]
			uptime := getUptimePercent(sla, mem, svc)
			billed := baseCost * (uptime / 100.0)

			pdf.CellFormat(colServiceW, rowH, svc, "1", 0, "L", false, 0, "")
			pdf.CellFormat(colBaseW, rowH, fmt.Sprintf("$%.2f", baseCost),
				"1", 0, "R", false, 0, "")
			pdf.CellFormat(colUptimeW, rowH, fmt.Sprintf("%.4f", uptime),
				"1", 0, "R", false, 0, "")
			pdf.CellFormat(colBillW, rowH, fmt.Sprintf("$%.2f", billed),
				"1", 1, "R", false, 0, "")

			memberTotal += billed
			grandTotal += billed
		}

		// subtotal
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(colServiceW+colBaseW+colUptimeW, rowH, "Member Total",
			"1", 0, "R", false, 0, "")
		pdf.CellFormat(colBillW, rowH, fmt.Sprintf("$%.2f", memberTotal),
			"1", 1, "R", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)

		// draw border around member box
		endY := pdf.GetY()
		pdf.Rect(leftMargin, startY-1, boxWidth, endY-startY+1, "D")

		pdf.Ln(boxGap)
	}

	// grand total (separate box)
	if pdf.GetY() > 260 {
		addPageWithWatermark(pdf, logoPath)
	}
	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(colServiceW+colBaseW+colUptimeW, rowH+1, "Grand Total",
		"1", 0, "R", false, 0, "")
	pdf.CellFormat(colBillW, rowH+1, fmt.Sprintf("$%.2f", grandTotal),
		"1", 1, "R", false, 0, "")

	filename := filepath.Join(tmpDir,
		fmt.Sprintf("member_billing_%s.pdf", month.Format("200601")))
	if err := pdf.OutputFileAndClose(filename); err != nil {
		return err
	}
	log.Log(log.Info, "[billing] member‑billing PDF written → %s", filename)
	return nil
}

/* --------------------------------------------------------------------- */

func Version() string { return "v0.4.4" }
