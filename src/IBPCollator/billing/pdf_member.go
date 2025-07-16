package billing

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	cfg "ibp-geodns/src/common/config"
	data2 "ibp-geodns/src/common/data2"
	log "ibp-geodns/src/common/logging"

	"github.com/phpdave11/gofpdf"
)

// writeMemberPDF generates an individual PDF for a member
func writeMemberPDF(memberName string, sum *Summary, sla SLASummary, outDir string, month time.Time) error {
	c := cfg.GetConfig()
	logoPath := findLogo(filepath.Dir(outDir))

	filename := filepath.Join(outDir, fmt.Sprintf("%s-IBP-Service_%s.pdf",
		month.Format("2006_01"), sanitizeFilename(memberName)))

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetTitle(fmt.Sprintf("IBP Service Report - %s", memberName), false)
	pdf.SetAuthor("IBPCollator "+Version(), false)

	// Custom header with modern design
	pdf.SetHeaderFuncMode(func() {
		pdf.SetFillColor(30, 30, 30)
		pdf.Rect(0, 0, 210, 25, "F")

		// Logo space
		if logoPath != "" {
			pdf.Image(logoPath, 10, 5, 30, 0, false, "", 0, "")
		}

		pdf.SetTextColor(255, 255, 255)
		pdf.SetFont("Helvetica", "B", 16)
		pdf.SetXY(50, 8)
		pdf.CellFormat(100, 8, "IBP Network Service Report", "", 0, "L", false, 0, "")

		pdf.SetFont("Helvetica", "", 10)
		pdf.SetXY(50, 16)
		pdf.CellFormat(100, 5, month.Format("January 2006"), "", 0, "L", false, 0, "")

		// Member name on right
		pdf.SetFont("Helvetica", "B", 12)
		pdf.SetXY(150, 10)
		pdf.CellFormat(50, 8, memberName, "", 0, "R", false, 0, "")

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

	// Member info section with modern card design
	memberConfig, hasMemberConfig := c.Members[memberName]
	memberCost := sum.Members[memberName]
	stats := calculateMemberStats(month)[memberName]
	totalRequests := calculateTotalRequests(month)

	// Member details card
	drawCard(pdf, 10, 35, 190, 40)

	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetXY(15, 40)
	pdf.CellFormat(180, 8, "Member Information", "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 10)
	y := 50.0

	if hasMemberConfig {
		if memberConfig.Details.Website != "" {
			pdf.SetXY(15, y)
			pdf.CellFormat(40, 5, "Website:", "", 0, "L", false, 0, "")
			pdf.SetFont("Helvetica", "B", 10)
			pdf.CellFormat(140, 5, memberConfig.Details.Website, "", 1, "L", false, 0, "")
			pdf.SetFont("Helvetica", "", 10)
			y += 6
		}

		pdf.SetXY(15, y)
		pdf.CellFormat(40, 5, "Member Level:", "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(40, 5, fmt.Sprintf("%d", memberConfig.Membership.Level), "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)

		pdf.SetXY(105, y)
		pdf.CellFormat(40, 5, "Member Since:", "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "B", 10)
		joinedTime := time.Unix(int64(memberConfig.Membership.Joined), 0)
		pdf.CellFormat(40, 5, joinedTime.Format("January 2006"), "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		y += 6
	}

	// Usage statistics card
	y = 85
	drawCard(pdf, 10, y, 190, 35)

	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetXY(15, y+5)
	pdf.CellFormat(180, 8, "Usage Statistics", "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 10)
	pdf.SetXY(15, y+15)
	pdf.CellFormat(40, 5, "DNS Requests:", "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(40, 5, fmt.Sprintf("%d", stats.RequestCount), "", 0, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 10)
	pdf.SetXY(105, y+15)
	pdf.CellFormat(40, 5, "% of Network:", "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "B", 10)
	percentage := 0.0
	if totalRequests > 0 {
		percentage = float64(stats.RequestCount) / float64(totalRequests) * 100.0
	}
	pdf.CellFormat(40, 5, fmt.Sprintf("%.2f%%", percentage), "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)

	// Service details
	y = 130
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetXY(10, y)
	pdf.CellFormat(190, 8, "Service Details", "", 1, "L", false, 0, "")
	y += 10

	// Sort services
	svcNames := make([]string, 0, len(memberCost.ServiceCosts))
	for s := range memberCost.ServiceCosts {
		svcNames = append(svcNames, s)
	}
	sort.Strings(svcNames)

	memberTotal := 0.0
	for _, svcName := range svcNames {
		if y > 250 {
			pdf.AddPage()
			y = 35
		}

		// Service card
		drawCard(pdf, 10, y, 190, 45)

		// Service header
		pdf.SetFillColor(240, 240, 240)
		pdf.Rect(10, y, 190, 10, "F")

		pdf.SetFont("Helvetica", "B", 11)
		pdf.SetXY(15, y+2)
		pdf.CellFormat(180, 6, svcName, "", 1, "L", false, 0, "")

		baseCost := memberCost.ServiceCosts[svcName]
		breakdown := getSLABreakdown(sla, memberName, svcName)
		billed := baseCost * (breakdown.Uptime / 100.0)
		memberTotal += billed

		// Service details
		pdf.SetFont("Helvetica", "", 9)
		y += 12

		// Resources
		if svcConfig, exists := c.Services[svcName]; exists {
			pdf.SetXY(15, y)
			pdf.SetTextColor(100, 100, 100)
			resourceText := fmt.Sprintf("Resources: %d nodes, %.1f cores, %.1f GB RAM, %.1f GB disk, %.1f TB bandwidth",
				svcConfig.Resources.Nodes,
				svcConfig.Resources.Cores,
				svcConfig.Resources.Memory,
				svcConfig.Resources.Disk,
				svcConfig.Resources.Bandwidth)
			pdf.CellFormat(180, 4, resourceText, "", 1, "L", false, 0, "")
			pdf.SetTextColor(0, 0, 0)
			y += 5
		}

		// Cost breakdown
		pdf.SetXY(15, y)
		pdf.CellFormat(40, 5, "Base Cost:", "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(30, 5, fmt.Sprintf("$%.2f", baseCost), "", 0, "R", false, 0, "")

		pdf.SetFont("Helvetica", "", 9)
		pdf.SetXY(85, y)
		pdf.CellFormat(40, 5, "Uptime:", "", 0, "L", false, 0, "")

		if breakdown.Uptime < DefaultSLAPercentage {
			pdf.SetTextColor(255, 0, 0)
		} else {
			pdf.SetTextColor(0, 128, 0)
		}
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(30, 5, fmt.Sprintf("%.2f%%", breakdown.Uptime), "", 0, "R", false, 0, "")
		pdf.SetTextColor(0, 0, 0)

		pdf.SetFont("Helvetica", "", 9)
		pdf.SetXY(155, y)
		pdf.CellFormat(25, 5, "Billed:", "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(25, 5, fmt.Sprintf("$%.2f", billed), "", 0, "R", false, 0, "")
		y += 7

		// SLA status
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetXY(15, y)
		if breakdown.MeetsSLA {
			pdf.SetTextColor(0, 128, 0)
			pdf.CellFormat(180, 4, fmt.Sprintf("✓ Meets SLA requirement of %.2f%%", DefaultSLAPercentage), "", 1, "L", false, 0, "")
		} else {
			pdf.SetTextColor(255, 0, 0)
			pdf.CellFormat(180, 4, fmt.Sprintf("✗ Below SLA: %.2f hours downtime (%.2f%% uptime required)",
				breakdown.HoursDown, DefaultSLAPercentage), "", 1, "L", false, 0, "")
		}
		pdf.SetTextColor(0, 0, 0)

		y += 15
	}

	// Total summary
	if y > 240 {
		pdf.AddPage()
		y = 35
	}

	drawCard(pdf, 10, y, 190, 20)
	pdf.SetFillColor(30, 30, 30)
	pdf.Rect(10, y, 190, 20, "F")

	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 12)
	pdf.SetXY(15, y+7)
	pdf.CellFormat(140, 6, "Total Amount Due", "", 0, "L", false, 0, "")
	pdf.CellFormat(40, 6, fmt.Sprintf("$%.2f", memberTotal), "", 0, "R", false, 0, "")
	pdf.SetTextColor(0, 0, 0)

	// Downtime events section
	pdf.AddPage()
	y = 35

	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(190, 8, "Downtime Events", "", 1, "L", false, 0, "")
	y += 10

	events := getMemberDowntimeEvents(memberName, month)
	if len(events) == 0 {
		drawCard(pdf, 10, y, 190, 20)
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetXY(15, y+7)
		pdf.SetTextColor(0, 128, 0)
		pdf.CellFormat(180, 6, "No downtime events recorded this month", "", 0, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	} else {
		for _, event := range events {
			if y > 250 {
				pdf.AddPage()
				y = 35
			}

			// Event card
			cardHeight := 35.0
			if event.VoteData != "" {
				cardHeight += 10
			}
			drawCard(pdf, 10, y, 190, cardHeight)

			// Event type and service
			pdf.SetFont("Helvetica", "B", 10)
			pdf.SetXY(15, y+5)
			serviceAffected := getServiceFromEvent(event)
			pdf.CellFormat(180, 5, fmt.Sprintf("%s - %s", strings.Title(event.CheckType), serviceAffected), "", 1, "L", false, 0, "")

			// Downtime period
			pdf.SetFont("Helvetica", "", 9)
			pdf.SetXY(15, y+12)
			duration := event.EndTime.Sub(event.StartTime)
			pdf.CellFormat(180, 4, fmt.Sprintf("Duration: %s (%.2f hours)",
				formatDuration(duration), duration.Hours()), "", 1, "L", false, 0, "")

			// Dates
			pdf.SetXY(15, y+18)
			pdf.SetTextColor(100, 100, 100)
			pdf.CellFormat(180, 4, fmt.Sprintf("From: %s",
				event.StartTime.Format("Jan 2, 2006 15:04 UTC")), "", 1, "L", false, 0, "")
			pdf.SetXY(15, y+23)
			pdf.CellFormat(180, 4, fmt.Sprintf("To: %s",
				event.EndTime.Format("Jan 2, 2006 15:04 UTC")), "", 1, "L", false, 0, "")
			pdf.SetTextColor(0, 0, 0)

			// Error and vote data
			if event.ErrorText != "" {
				pdf.SetXY(15, y+29)
				pdf.SetTextColor(200, 0, 0)
				pdf.SetFont("Helvetica", "", 8)
				pdf.CellFormat(180, 4, fmt.Sprintf("Error: %s", event.ErrorText), "", 1, "L", false, 0, "")
				pdf.SetTextColor(0, 0, 0)
			}

			if event.VoteData != "" {
				pdf.SetXY(15, y+34)
				pdf.SetFont("Helvetica", "", 8)
				pdf.SetTextColor(100, 100, 100)
				pdf.CellFormat(180, 4, fmt.Sprintf("Monitor votes: %s", event.VoteData), "", 1, "L", false, 0, "")
				pdf.SetTextColor(0, 0, 0)
			}

			y += cardHeight + 5
		}
	}

	if err := pdf.OutputFileAndClose(filename); err != nil {
		return err
	}

	log.Log(log.Info, "[billing] Member PDF written → %s", filename)
	return nil
}

// Helper functions
func drawCard(pdf *gofpdf.Fpdf, x, y, w, h float64) {
	pdf.SetDrawColor(200, 200, 200)
	pdf.SetLineWidth(0.3)
	pdf.Rect(x, y, w, h, "D")
	pdf.SetDrawColor(0, 0, 0)
}

func sanitizeFilename(name string) string {
	// Replace any characters that might cause issues in filenames
	replacer := strings.NewReplacer(
		" ", "_",
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

// getMemberDowntimeEvents retrieves downtime events for a member in the given month
func getMemberDowntimeEvents(memberName string, month time.Time) []DowntimeEvent {
	events := []DowntimeEvent{}

	if data2.DB == nil {
		return events
	}

	startTime := month
	endTime := month.AddDate(0, 1, 0).Add(-time.Second)

	query := `
		SELECT 
			check_type,
			check_name,
			COALESCE(domain_name, '') as domain_name,
			COALESCE(endpoint, '') as endpoint,
			start_time,
			COALESCE(end_time, ?) as end_time,
			COALESCE(error, '') as error,
			COALESCE(vote_data, '') as vote_data,
			is_ipv6
		FROM member_events
		WHERE member_name = ?
		AND status = 0
		AND start_time < ?
		AND (end_time IS NULL OR end_time > ?)
		ORDER BY start_time DESC
	`

	rows, err := data2.DB.Query(query, endTime, memberName, endTime, startTime)
	if err != nil {
		log.Log(log.Error, "[billing] Failed to query downtime events: %v", err)
		return events
	}
	defer rows.Close()

	for rows.Next() {
		var event DowntimeEvent
		err := rows.Scan(
			&event.CheckType,
			&event.CheckName,
			&event.DomainName,
			&event.Endpoint,
			&event.StartTime,
			&event.EndTime,
			&event.ErrorText,
			&event.VoteData,
			&event.IsIPv6,
		)
		if err != nil {
			log.Log(log.Error, "[billing] Failed to scan downtime event: %v", err)
			continue
		}

		// Adjust times to be within month
		if event.StartTime.Before(startTime) {
			event.StartTime = startTime
		}
		if event.EndTime.After(endTime) {
			event.EndTime = endTime
		}

		events = append(events, event)
	}

	return events
}

func getServiceFromEvent(event DowntimeEvent) string {
	if event.CheckType == "site" {
		return "All Services"
	}

	// Map domain to service
	c := cfg.GetConfig()
	for svcName, svc := range c.Services {
		for _, provider := range svc.Providers {
			for _, rpcUrl := range provider.RpcUrls {
				if strings.Contains(rpcUrl, event.DomainName) {
					return svcName
				}
			}
		}
	}

	return event.DomainName
}

// DowntimeEvent represents a downtime event
type DowntimeEvent struct {
	CheckType  string
	CheckName  string
	DomainName string
	Endpoint   string
	StartTime  time.Time
	EndTime    time.Time
	ErrorText  string
	VoteData   string
	IsIPv6     bool
}

// createMonthlyZip creates a zip file of all PDFs for the month
func createMonthlyZip(monthDir, zipPath string) error {
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Walk through the directory and add files to zip
	err = filepath.Walk(monthDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only include PDF files
		if !strings.HasSuffix(strings.ToLower(path), ".pdf") {
			return nil
		}

		// Create relative path for the zip
		relPath, err := filepath.Rel(monthDir, path)
		if err != nil {
			return err
		}

		// Create file in zip
		zipFile, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		// Open source file
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// Copy file to zip
		_, err = io.Copy(zipFile, file)
		return err
	})

	return err
}
