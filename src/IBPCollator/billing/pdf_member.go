package billing

import (
	"fmt"
	"io"
	"net/http"
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

// downloadMemberLogo downloads a member's logo to the tmp/member_logos directory
func downloadMemberLogo(memberName, logoURL, baseDir string) string {
	if logoURL == "" {
		return ""
	}

	// Create member_logos directory
	logoDir := filepath.Join(baseDir, "tmp", "member_logos")
	if err := os.MkdirAll(logoDir, 0755); err != nil {
		log.Log(log.Error, "[billing] Failed to create logo directory: %v", err)
		return ""
	}

	// Sanitize filename
	filename := sanitizeFilename(memberName) + ".png"
	logoPath := filepath.Join(logoDir, filename)

	// Check if already downloaded
	if _, err := os.Stat(logoPath); err == nil {
		return logoPath
	}

	// Download the logo
	resp, err := http.Get(logoURL)
	if err != nil {
		log.Log(log.Error, "[billing] Failed to download logo for %s: %v", memberName, err)
		return ""
	}
	defer resp.Body.Close()

	// Create the file
	file, err := os.Create(logoPath)
	if err != nil {
		log.Log(log.Error, "[billing] Failed to create logo file for %s: %v", memberName, err)
		return ""
	}
	defer file.Close()

	// Copy the logo
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		log.Log(log.Error, "[billing] Failed to save logo for %s: %v", memberName, err)
		os.Remove(logoPath)
		return ""
	}

	return logoPath
}

// groupServicesByLevel groups services by their level requirement
func groupServicesByLevel(memberCost MemberCost, services map[string]cfg.Service) map[int][]string {
	levelGroups := make(map[int][]string)

	for svcName := range memberCost.ServiceCosts {
		if svc, exists := services[svcName]; exists {
			level := svc.Configuration.LevelRequired
			levelGroups[level] = append(levelGroups[level], svcName)
		}
	}

	// Sort services within each level
	for level := range levelGroups {
		sort.Strings(levelGroups[level])
	}

	return levelGroups
}

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

	// Get member configuration
	memberConfig, hasMemberConfig := c.Members[memberName]
	memberCost := sum.Members[memberName]

	// Use the member's Details.Name for database lookup
	dbMemberName := memberName
	if hasMemberConfig && memberConfig.Details.Name != "" {
		dbMemberName = memberConfig.Details.Name
	}

	stats := calculateMemberStats(month)[dbMemberName]
	totalRequests := calculateTotalRequests(month)

	// Download member logo
	memberLogoPath := ""
	if hasMemberConfig && memberConfig.Details.Logo != "" {
		memberLogoPath = downloadMemberLogo(memberName, memberConfig.Details.Logo, c.Local.System.WorkDir)
	}

	// Member information card - reduced height
	drawMemberCard(pdf, 10, 35, 190, 60)
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetXY(15, 40)
	pdf.CellFormat(120, 8, "Member Information", "", 1, "L", false, 0, "")

	// Member logo on the right
	if memberLogoPath != "" {
		info := pdf.RegisterImageOptions(memberLogoPath,
			gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true})
		if info != nil {
			logoW, logoH := 40.0, 40.0
			aspectRatio := info.Width() / info.Height()
			if aspectRatio > 1 {
				logoH = logoW / aspectRatio
			} else {
				logoW = logoH * aspectRatio
			}
			pdf.ImageOptions(memberLogoPath, 155, 50, logoW, logoH,
				false, gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}, 0, "")
		}
	}

	pdf.SetFont("Helvetica", "", 10)
	y := 50.0

	// Left column
	if hasMemberConfig {
		// Website
		if memberConfig.Details.Website != "" {
			pdf.SetXY(15, y)
			pdf.CellFormat(30, 5, "Website:", "", 0, "L", false, 0, "")
			pdf.SetX(45)
			pdf.SetFont("Helvetica", "B", 10)
			pdf.CellFormat(100, 5, memberConfig.Details.Website, "", 1, "L", false, 0, "")
			pdf.SetFont("Helvetica", "", 10)
			y += 6
		}

		// Member Level and Since
		pdf.SetXY(15, y)
		pdf.CellFormat(30, 5, "Member Level:", "", 0, "L", false, 0, "")
		pdf.SetX(45)
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(30, 5, fmt.Sprintf("%d", memberConfig.Membership.Level), "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)

		pdf.SetXY(80, y)
		pdf.CellFormat(30, 5, "Since:", "", 0, "L", false, 0, "")
		pdf.SetX(95)
		pdf.SetFont("Helvetica", "B", 10)
		joinedTime := time.Unix(int64(memberConfig.Membership.Joined), 0)
		pdf.CellFormat(40, 5, joinedTime.Format("Jan 2006"), "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		y += 6

		// Location info
		pdf.SetXY(15, y)
		pdf.CellFormat(30, 5, "Region:", "", 0, "L", false, 0, "")
		pdf.SetX(45)
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(100, 5, memberConfig.Location.Region, "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		y += 6

		// Coordinates
		pdf.SetXY(15, y)
		pdf.CellFormat(30, 5, "Coordinates:", "", 0, "L", false, 0, "")
		pdf.SetX(45)
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(100, 5, fmt.Sprintf("%.4f, %.4f", memberConfig.Location.Latitude, memberConfig.Location.Longitude), "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		y += 6

		// Service IPs - Always show both IPv4 and IPv6
		pdf.SetXY(15, y)
		pdf.CellFormat(30, 5, "IPv4:", "", 0, "L", false, 0, "")
		pdf.SetX(45)
		pdf.SetFont("Helvetica", "B", 10)
		if memberConfig.Service.ServiceIPv4 != "" {
			pdf.CellFormat(100, 5, memberConfig.Service.ServiceIPv4, "", 1, "L", false, 0, "")
		} else {
			pdf.CellFormat(100, 5, "", "", 1, "L", false, 0, "")
		}
		pdf.SetFont("Helvetica", "", 10)
		y += 6

		pdf.SetXY(15, y)
		pdf.CellFormat(30, 5, "IPv6:", "", 0, "L", false, 0, "")
		pdf.SetX(45)
		pdf.SetFont("Helvetica", "B", 10)
		if memberConfig.Service.ServiceIPv6 != "" {
			pdf.CellFormat(100, 5, memberConfig.Service.ServiceIPv6, "", 1, "L", false, 0, "")
		} else {
			pdf.CellFormat(100, 5, "", "", 1, "L", false, 0, "")
		}
		pdf.SetFont("Helvetica", "", 10)
		y += 6
	}

	// DNS usage statistics - moved up with consistent spacing
	pdf.SetXY(15, y)
	pdf.CellFormat(30, 5, "DNS Requests:", "", 0, "L", false, 0, "")
	pdf.SetX(45)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(30, 5, fmt.Sprintf("%d", stats.RequestCount), "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)

	pdf.SetXY(80, y)
	pdf.CellFormat(30, 5, "% of Network:", "", 0, "L", false, 0, "")
	pdf.SetX(110)
	pdf.SetFont("Helvetica", "B", 10)
	percentage := 0.0
	if totalRequests > 0 {
		percentage = float64(stats.RequestCount) / float64(totalRequests) * 100.0
	}
	pdf.CellFormat(30, 5, fmt.Sprintf("%.2f%%", percentage), "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	y += 6

	// Create separate Overview box below member information - reduced height
	y = 100
	drawMemberCard(pdf, 10, y, 190, 46)
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetXY(15, y+5)
	pdf.CellFormat(100, 8, "Overview", "", 1, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 10)
	y += 15

	// Calculate totals for overview
	totalBilled := 0.0
	totalServices := 0
	totalDowntimeHours := 0.0
	totalServiceHours := 0.0
	totalCores := 0.0
	totalMemory := 0.0
	totalDisk := 0.0
	totalBandwidth := 0.0

	for svcName, baseCost := range memberCost.ServiceCosts {
		totalServices++
		breakdown := getSLABreakdown(sla, memberName, svcName)
		billed := baseCost * (breakdown.Uptime / 100.0)
		totalBilled += billed
		totalDowntimeHours += breakdown.HoursDown
		totalServiceHours += breakdown.HoursTotal

		// Get resource totals
		if svcConfig, exists := c.Services[svcName]; exists {
			totalCores += svcConfig.Resources.Cores * float64(svcConfig.Resources.Nodes)
			totalMemory += svcConfig.Resources.Memory * float64(svcConfig.Resources.Nodes)
			totalDisk += svcConfig.Resources.Disk * float64(svcConfig.Resources.Nodes)
			totalBandwidth += svcConfig.Resources.Bandwidth * float64(svcConfig.Resources.Nodes)
		}
	}

	// Calculate overall uptime percentage
	totalUptime := 100.0
	if totalServiceHours > 0 {
		totalUptime = ((totalServiceHours - totalDowntimeHours) / totalServiceHours) * 100.0
	}

	// Calculate base total for the member
	memberBaseTotal := 0.0
	for _, baseCost := range memberCost.ServiceCosts {
		memberBaseTotal += baseCost
	}

	slaPenalty := memberBaseTotal - totalBilled

	// First row - financial
	pdf.SetXY(15, y)
	pdf.CellFormat(35, 5, "Total Payment:", "", 0, "L", false, 0, "")
	pdf.SetX(50)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(30, 5, fmt.Sprintf("$%.2f", totalBilled), "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)

	pdf.SetXY(80, y)
	pdf.CellFormat(35, 5, "SLA Credits:", "", 0, "L", false, 0, "")
	pdf.SetX(115)
	if slaPenalty > 0 {
		pdf.SetTextColor(0, 150, 0)
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(30, 5, fmt.Sprintf("-$%.2f", slaPenalty), "", 1, "L", false, 0, "")
	} else {
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(30, 5, "$0.00", "", 1, "L", false, 0, "")
	}
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Helvetica", "", 10)
	y += 6

	// Second row - services and uptime
	pdf.SetXY(15, y)
	pdf.CellFormat(35, 5, "Total Services:", "", 0, "L", false, 0, "")
	pdf.SetX(50)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(30, 5, fmt.Sprintf("%d", totalServices), "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)

	pdf.SetXY(80, y)
	pdf.CellFormat(35, 5, "Avg Uptime:", "", 0, "L", false, 0, "")
	pdf.SetX(115)
	if totalUptime < DefaultSLAPercentage {
		pdf.SetTextColor(255, 0, 0)
	} else {
		pdf.SetTextColor(0, 150, 0)
	}
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(30, 5, fmt.Sprintf("%.2f%%", totalUptime), "", 1, "L", false, 0, "")
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Helvetica", "", 10)
	y += 6

	// Resources header
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetXY(15, y)
	pdf.CellFormat(100, 5, "Total Resources:", "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	y += 5

	// Resources row 1
	pdf.SetXY(15, y)
	pdf.CellFormat(25, 5, "Cores:", "", 0, "L", false, 0, "")
	pdf.SetX(40)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(30, 5, fmt.Sprintf("%.1f", totalCores), "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)

	pdf.SetXY(80, y)
	pdf.CellFormat(25, 5, "Memory:", "", 0, "L", false, 0, "")
	pdf.SetX(105)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(30, 5, fmt.Sprintf("%.1f GB", totalMemory), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	y += 5

	// Resources row 2
	pdf.SetXY(15, y)
	pdf.CellFormat(25, 5, "Disk:", "", 0, "L", false, 0, "")
	pdf.SetX(40)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(30, 5, fmt.Sprintf("%.1f GB", totalDisk), "", 0, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)

	pdf.SetXY(80, y)
	pdf.CellFormat(25, 5, "Bandwidth:", "", 0, "L", false, 0, "")
	pdf.SetX(105)
	pdf.SetFont("Helvetica", "B", 10)
	pdf.CellFormat(40, 5, fmt.Sprintf("%.1f GB", totalBandwidth), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 10)

	// Service details grouped by level
	y = 155
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetXY(10, y)
	pdf.CellFormat(190, 8, "Service Details", "", 1, "L", false, 0, "")
	y += 10

	// Group services by level
	levelGroups := groupServicesByLevel(memberCost, c.Services)

	// Get sorted levels (ascending)
	levels := make([]int, 0, len(levelGroups))
	for level := range levelGroups {
		levels = append(levels, level)
	}
	sort.Ints(levels)

	memberTotal := 0.0

	// Process each level group
	for _, level := range levels {
		services := levelGroups[level]
		if len(services) == 0 {
			continue
		}

		if y > 230 {
			pdf.AddPage()
			y = 35
		}

		// Level header
		pdf.SetFont("Helvetica", "B", 14)
		pdf.SetXY(10, y-2)
		pdf.CellFormat(190, 7, fmt.Sprintf("Level %d Services", level), "", 1, "L", false, 0, "")
		y += 8

		levelTotal := 0.0

		// Process services in this level
		for _, svcName := range services {
			// Calculate service card height based on downtime events
			baseHeight := 33.0
			events := getServiceDowntimeEvents(dbMemberName, svcName, month)
			filteredEvents := filterEvents(events, 5) // 5+ minute events
			if len(filteredEvents) > 0 {
				baseHeight += 8 + float64(len(filteredEvents))*6 // Header + rows
			}

			if y+baseHeight > 270 {
				pdf.AddPage()
				y = 35
			}

			// Service card
			drawMemberCard(pdf, 10, y, 190, baseHeight)

			// Service header
			pdf.SetFillColor(240, 240, 240)
			pdf.Rect(10, y, 190, 10, "F")
			pdf.SetFont("Helvetica", "B", 11)
			pdf.SetXY(15, y+2)
			pdf.CellFormat(180, 6, svcName, "", 1, "L", false, 0, "")

			baseCost := memberCost.ServiceCosts[svcName]
			breakdown := getSLABreakdown(sla, memberName, svcName)
			billed := baseCost * (breakdown.Uptime / 100.0)
			levelTotal += billed
			memberTotal += billed

			// Service details
			pdf.SetFont("Helvetica", "", 9)
			serviceY := y + 12

			// Resources
			if svcConfig, exists := c.Services[svcName]; exists {
				pdf.SetXY(15, serviceY)
				pdf.SetTextColor(100, 100, 100)
				resourceText := fmt.Sprintf("Resources: %d nodes, %.1f cores, %.1f GB RAM, %.1f GB disk, %.1f GB bandwidth",
					svcConfig.Resources.Nodes,
					svcConfig.Resources.Cores,
					svcConfig.Resources.Memory,
					svcConfig.Resources.Disk,
					svcConfig.Resources.Bandwidth)
				pdf.CellFormat(180, 4, resourceText, "", 1, "L", false, 0, "")
				pdf.SetTextColor(0, 0, 0)
				serviceY += 5
			}

			// Cost breakdown
			pdf.SetXY(15, serviceY)
			pdf.CellFormat(25, 5, "Base Cost:", "", 0, "L", false, 0, "")
			pdf.SetX(40)
			pdf.SetFont("Helvetica", "B", 9)
			pdf.CellFormat(20, 5, fmt.Sprintf("$%.2f", baseCost), "", 0, "R", false, 0, "")
			pdf.SetFont("Helvetica", "", 9)

			pdf.SetX(70)
			pdf.CellFormat(20, 5, "Uptime:", "", 0, "L", false, 0, "")
			pdf.SetX(90)
			if breakdown.Uptime < DefaultSLAPercentage {
				pdf.SetTextColor(255, 0, 0)
			} else {
				pdf.SetTextColor(0, 128, 0)
			}
			pdf.SetFont("Helvetica", "B", 9)
			pdf.CellFormat(25, 5, fmt.Sprintf("%.2f%%", breakdown.Uptime), "", 0, "R", false, 0, "")
			pdf.SetTextColor(0, 0, 0)
			pdf.SetFont("Helvetica", "", 9)

			pdf.SetX(125)
			pdf.CellFormat(20, 5, "Billed:", "", 0, "L", false, 0, "")
			pdf.SetX(145)
			pdf.SetFont("Helvetica", "B", 9)
			pdf.CellFormat(25, 5, fmt.Sprintf("$%.2f", billed), "", 0, "R", false, 0, "")

			serviceY += 7

			// SLA status
			pdf.SetFont("Helvetica", "", 9)
			pdf.SetXY(15, serviceY)
			if breakdown.MeetsSLA {
				pdf.SetTextColor(0, 128, 0)
				pdf.CellFormat(180, 4, fmt.Sprintf("[OK] Meets SLA requirement of %.2f%%", DefaultSLAPercentage), "", 1, "L", false, 0, "")
			} else {
				pdf.SetTextColor(255, 0, 0)
				pdf.CellFormat(180, 4, fmt.Sprintf("[FAIL] Below SLA: %.2f hours downtime (%.2f%% uptime required)",
					breakdown.HoursDown, DefaultSLAPercentage), "", 1, "L", false, 0, "")
			}
			pdf.SetTextColor(0, 0, 0)
			serviceY += 6

			// Downtime table for this service (if any)
			if len(filteredEvents) > 0 {
				pdf.SetDrawColor(200, 200, 200)
				pdf.Line(15, serviceY, 195, serviceY)
				pdf.SetDrawColor(0, 0, 0)
				serviceY += 3

				pdf.SetFont("Helvetica", "B", 8)
				pdf.SetXY(15, serviceY)
				pdf.CellFormat(180, 4, "Downtime Events (5+ minutes):", "", 1, "L", false, 0, "")
				serviceY += 5

				// Table header
				pdf.SetFont("Helvetica", "", 7)
				pdf.SetFillColor(245, 245, 245)
				pdf.SetXY(15, serviceY)
				pdf.CellFormat(25, 4, "Duration", "1", 0, "L", true, 0, "")
				pdf.CellFormat(50, 4, "Start Time", "1", 0, "L", true, 0, "")
				pdf.CellFormat(50, 4, "End Time", "1", 0, "L", true, 0, "")
				pdf.CellFormat(55, 4, "Error", "1", 1, "L", true, 0, "")
				serviceY += 4

				// Downtime rows
				for _, event := range filteredEvents {
					duration := event.EndTime.Sub(event.StartTime)
					pdf.SetXY(15, serviceY)
					pdf.SetFont("Helvetica", "", 6)
					pdf.CellFormat(25, 4, formatDuration(duration), "1", 0, "L", false, 0, "")
					pdf.CellFormat(50, 4, event.StartTime.Format("Jan 2 15:04 UTC"), "1", 0, "L", false, 0, "")
					pdf.CellFormat(50, 4, event.EndTime.Format("Jan 2 15:04 UTC"), "1", 0, "L", false, 0, "")
					errorText := event.ErrorText
					if len(errorText) > 40 {
						errorText = errorText[:37] + "..."
					}
					pdf.CellFormat(55, 4, errorText, "1", 1, "L", false, 0, "")
					serviceY += 4
				}
			}

			y = serviceY + 12
		}

		// Level total
		if len(services) > 1 {
			if y > 260 {
				pdf.AddPage()
				y = 35
			}
			pdf.SetFont("Helvetica", "B", 14)
			pdf.SetXY(110, y-2)
			pdf.CellFormat(60, 6, fmt.Sprintf("Level %d Total:", level), "", 0, "R", false, 0, "")
			pdf.CellFormat(30, 6, fmt.Sprintf("$%.2f", levelTotal), "", 1, "R", false, 0, "")
			y += 12
		}
	}

	// Total summary
	if y > 240 {
		pdf.AddPage()
		y = 35
	}

	drawMemberCard(pdf, 10, y, 190, 20)
	pdf.SetFillColor(30, 30, 30)
	pdf.Rect(10, y, 190, 20, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetXY(15, y+7)
	pdf.CellFormat(140, 6, "Total Amount Due (All Services)", "", 0, "L", false, 0, "")
	pdf.CellFormat(35, 6, fmt.Sprintf("$%.2f", memberTotal), "", 0, "R", false, 0, "")
	pdf.SetTextColor(0, 0, 0)

	if err := pdf.OutputFileAndClose(filename); err != nil {
		return err
	}

	log.Log(log.Info, "[billing] Member PDF written → %s", filename)
	return nil
}

// getServiceDowntimeEvents retrieves downtime events for a specific service
func getServiceDowntimeEvents(memberName, serviceName string, month time.Time) []DowntimeEvent {
	events := []DowntimeEvent{}
	if data2.DB == nil {
		return events
	}

	// Map service to domains
	c := cfg.GetConfig()
	domains := []string{}
	if svc, exists := c.Services[serviceName]; exists {
		for _, provider := range svc.Providers {
			for _, rpcUrl := range provider.RpcUrls {
				if domain := extractDomainFromURL(rpcUrl); domain != "" {
					domains = append(domains, domain)
				}
			}
		}
	}

	startTime := month
	endTime := month.AddDate(0, 1, 0).Add(-time.Nanosecond)

	// First, get site-level events (affects all services)
	siteQuery := `
		SELECT  
			check_type,
			check_name,
			COALESCE(domain_name, '') as domain_name,
			COALESCE(endpoint, '') as endpoint,
			start_time,
			end_time,
			COALESCE(error, '') as error,
			COALESCE(vote_data, '') as vote_data,
			is_ipv6
		FROM member_events
		WHERE member_name = ?
		AND check_type = 'site'
		AND status = 0
		AND (
			(start_time < ? AND (end_time IS NULL OR end_time > ?))
			OR
			(start_time >= ? AND start_time < ?)
		)
		ORDER BY start_time DESC
	`

	rows, err := data2.DB.Query(siteQuery, memberName, endTime, startTime, startTime, endTime)
	if err != nil {
		log.Log(log.Error, "[billing] Failed to query site downtime events: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var event DowntimeEvent
			var endTimePtr *time.Time
			err := rows.Scan(
				&event.CheckType,
				&event.CheckName,
				&event.DomainName,
				&event.Endpoint,
				&event.StartTime,
				&endTimePtr,
				&event.ErrorText,
				&event.VoteData,
				&event.IsIPv6,
			)
			if err != nil {
				log.Log(log.Error, "[billing] Failed to scan site downtime event: %v", err)
				continue
			}

			// Adjust times to be within month
			if event.StartTime.Before(startTime) {
				event.StartTime = startTime
			}

			if endTimePtr != nil {
				event.EndTime = *endTimePtr
				if event.EndTime.After(endTime) {
					event.EndTime = endTime
				}
			} else {
				event.EndTime = endTime
			}

			events = append(events, event)
		}
	}

	// Then get service-specific events
	if len(domains) > 0 {
		// Build parameterized query with proper placeholders
		placeholders := make([]string, len(domains))
		args := make([]interface{}, 0, len(domains)+5)
		args = append(args, memberName)
		for i, domain := range domains {
			placeholders[i] = "?"
			args = append(args, domain)
		}
		args = append(args, endTime, startTime, startTime, endTime)

		// Safe query construction with parameterized inputs
		query := fmt.Sprintf(`
			SELECT  
				check_type,
				check_name,
				COALESCE(domain_name, '') as domain_name,
				COALESCE(endpoint, '') as endpoint,
				start_time,
				end_time,
				COALESCE(error, '') as error,
				COALESCE(vote_data, '') as vote_data,
				is_ipv6
			FROM member_events
			WHERE member_name = ?
			AND check_type IN ('domain', 'endpoint')
			AND status = 0
			AND domain_name IN (%s)
			AND (
				(start_time < ? AND (end_time IS NULL OR end_time > ?))
				OR
				(start_time >= ? AND start_time < ?)
			)
			ORDER BY start_time DESC
		`, strings.Join(placeholders, ","))

		rows, err := data2.DB.Query(query, args...)
		if err != nil {
			log.Log(log.Error, "[billing] Failed to query service downtime events: %v", err)
			return events
		}
		defer rows.Close()

		for rows.Next() {
			var event DowntimeEvent
			var endTimePtr *time.Time
			err := rows.Scan(
				&event.CheckType,
				&event.CheckName,
				&event.DomainName,
				&event.Endpoint,
				&event.StartTime,
				&endTimePtr,
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

			if endTimePtr != nil {
				event.EndTime = *endTimePtr
				if event.EndTime.After(endTime) {
					event.EndTime = endTime
				}
			} else {
				event.EndTime = endTime
			}

			events = append(events, event)
		}
	}

	return events
}

// filterEvents filters events to only show those longer than minMinutes
func filterEvents(events []DowntimeEvent, minMinutes float64) []DowntimeEvent {
	filtered := []DowntimeEvent{}
	for _, event := range events {
		duration := event.EndTime.Sub(event.StartTime)
		if duration.Minutes() >= minMinutes {
			filtered = append(filtered, event)
		}
	}
	return filtered
}

// Helper functions remain the same...
func drawMemberCard(pdf *gofpdf.Fpdf, x, y, w, h float64) {
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
