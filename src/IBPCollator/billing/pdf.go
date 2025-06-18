package billing

// ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓
// ┃  Stake Plus Inc. – IBPCollator Billing PDF helpers  (v0.4.7)       ┃
// ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛
//
// Changes in this revision
// ------------------------
// • Watermark logo now scales to 62.5 % of page width (was 50 %).
// • Watermark transparency reduced → α = 0.25 (was 0.18) for higher opacity.
// • Report titles remain exclusively in page headers – never inside data boxes.
// • Retains build‑fix for gofpdf.GetMargins() (four return values).
//
// NOTE: No public API or data‑contract changes – safe drop‑in upgrade.

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"time"

	cfg "ibp-geodns/src/common/config"
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

// addPageWithWatermark draws one blank page, then renders the company logo
// centred, 62 ½ % page‑width, with 25 % transparency, and finally restores α.
func addPageWithWatermark(pdf *gofpdf.Fpdf, logo string) {
	pdf.AddPage()
	if logo == "" {
		return
	}
	pageW, pageH := pdf.GetPageSize()
	imgW := pageW * 0.625 // 62.5 % of page width (50 % + 25 %)

	info := pdf.RegisterImageOptions(logo,
		gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true})
	nativeW, nativeH := info.Extent()
	scale := imgW / nativeW
	imgH := nativeH * scale
	imgX := (pageW - imgW) / 2
	imgY := (pageH - imgH) / 2

	pdf.SetAlpha(0.25, "Normal") // 25 % transparency (more opaque)
	pdf.ImageOptions(logo, imgX, imgY, imgW, 0,
		false, gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, 0, "")
	pdf.SetAlpha(1, "Normal")
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
                     “cost by service”  –  PDF report
--------------------------------------------------------------------- */

func writeServiceCostPDF(sum *Summary, tmpDir string) error {
	c := cfg.GetConfig()
	logoPath := findLogo(c.Local.System.WorkDir)

	const title = "IBP Network - Cost by Service"

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(title, false)
	pdf.SetAuthor("IBPCollator "+Version(), false)

	// Global header – appears on every page, outside any service box
	pdf.SetHeaderFuncMode(func() {
		pdf.SetFont("Helvetica", "B", 15)
		pdf.CellFormat(0, 10, title, "", 1, "C", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.CellFormat(0, 6, time.Now().UTC().Format("02 Jan 2006 15:04 UTC"),
			"", 0, "C", false, 0, "")
		pdf.Ln(6)
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

	origLeft, _, _, _ := pdf.GetMargins() // four‑value signature

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

	const titleFmt = "IBP Network - Member Billing (%s %d)"
	title := fmt.Sprintf(titleFmt, month.Format("January"), month.Year())

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(title, false)
	pdf.SetAuthor("IBPCollator "+Version(), false)

	// Global header – appears on every page, outside any member box
	pdf.SetHeaderFuncMode(func() {
		pdf.SetFont("Helvetica", "B", 15)
		pdf.CellFormat(0, 10, title, "", 1, "C", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.CellFormat(0, 6, time.Now().UTC().Format("02 Jan 2006 15:04 UTC"),
			"", 0, "C", false, 0, "")
		pdf.Ln(6)
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
		colBillW    = 40.0
	)
	boxWidth := colServiceW + colBaseW + colUptimeW + colBillW
	leftMargin := (pageW - boxWidth) / 2

	origLeft, _, _, _ := pdf.GetMargins()

	grandTotal := 0.0

	for _, mem := range memberNames {
		startY := pdf.GetY()
		if startY > 210 {
			addPageWithWatermark(pdf, logoPath)
			startY = pdf.GetY()
		}

		pdf.SetLeftMargin(leftMargin)
		pdf.SetX(leftMargin)

		// Member title (now always just the member name)
		pdf.SetFont("Helvetica", "B", 12)
		pdf.CellFormat(boxWidth, rowH+3, mem, "", 1, "L", false, 0, "")

		// metadata lines
		pdf.SetFont("Helvetica", "", 9)
		metaLines := make([]string, 0, 6)

		if web, ok := lookupString(sum.Members[mem], "Website"); ok && web != "" {
			metaLines = append(metaLines, "Website: "+web)
		}
		metaLines = append(metaLines,
			"Billing period: "+month.Format("January 2006"))

		if lvl, ok := lookupString(sum.Members[mem], "Level"); ok && lvl != "" {
			metaLines = append(metaLines, "IBP member level: "+lvl)
		}

		if joined, ok := lookupString(sum.Members[mem], "Joined"); ok && joined != "" {
			metaLines = append(metaLines, "Joined: "+joined)
		}

		if req, ok := lookupInt64(sum.Members[mem], "DNSRequests"); ok && req > 0 {
			metaLines = append(metaLines,
				"DNS requests (period): "+strconv.FormatInt(req, 10))
		}

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
		pdf.CellFormat(colBillW, rowH, "Billed (USD)", "1", 1, "R", true, 0, "")
		pdf.SetFont("Helvetica", "", 10)

		memberTotal := 0.0
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

				// re‑header
				pdf.SetFont("Helvetica", "B", 11)
				pdf.CellFormat(colServiceW, rowH, "Service", "1", 0, "L", true, 0, "")
				pdf.CellFormat(colBaseW, rowH, "Base (USD)", "1", 0, "R", true, 0, "")
				pdf.CellFormat(colUptimeW, rowH, "Uptime %", "1", 0, "R", true, 0, "")
				pdf.CellFormat(colBillW, rowH, "Billed (USD)", "1", 1, "R", true, 0, "")
				pdf.SetFont("Helvetica", "", 10)
			}

			baseCost := sum.Members[mem].ServiceCosts[svc]
			uptime := getUptimePercent(sla, mem, svc)
			billed := baseCost * (uptime / 100.0)

			fillToggle = !fillToggle
			pdf.CellFormat(colServiceW, rowH, svc, "1", 0, "L", fillToggle, 0, "")
			pdf.CellFormat(colBaseW, rowH, fmt.Sprintf("$%.2f", baseCost),
				"1", 0, "R", fillToggle, 0, "")
			pdf.CellFormat(colUptimeW, rowH, fmt.Sprintf("%.4f", uptime),
				"1", 0, "R", fillToggle, 0, "")
			pdf.CellFormat(colBillW, rowH, fmt.Sprintf("$%.2f", billed),
				"1", 1, "R", fillToggle, 0, "")

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
	pdf.CellFormat(colServiceW+colBaseW+colUptimeW, rowH+1, "Grand Total",
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

/* --------------------------------------------------------------------- */

func Version() string { return "v0.4.7" }
