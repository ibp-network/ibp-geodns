package api

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
	log "ibp-geodns/src/common/logging"
)

// initPDFManager initializes the PDF manager
func initPDFManager() {
	c := cfg.GetConfig()
	baseDir := filepath.Join(c.Local.System.WorkDir, "tmp")

	pdfManager = &PDFManager{
		pdfFiles: make(map[string][]PDFInfo),
		baseDir:  baseDir,
	}

	// Initial scan
	pdfManager.scanPDFFiles()

	// Start periodic scanning
	go pdfManager.startPeriodicScan()
}

// startPeriodicScan runs a scan every 5 minutes
func (pm *PDFManager) startPeriodicScan() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		pm.scanPDFFiles()
	}
}

// scanPDFFiles scans the tmp directory for YYYY-MM folders containing PDFs
func (pm *PDFManager) scanPDFFiles() {
	log.Log(log.Debug, "[PDFManager] Starting PDF file scan in %s", pm.baseDir)

	newFiles := make(map[string][]PDFInfo)

	// Read the tmp directory
	entries, err := os.ReadDir(pm.baseDir)
	if err != nil {
		log.Log(log.Error, "[PDFManager] Failed to read directory %s: %v", pm.baseDir, err)
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Check if directory name matches YYYY-MM pattern
		if !monthDirPattern.MatchString(entry.Name()) {
			continue
		}

		monthDir := filepath.Join(pm.baseDir, entry.Name())
		monthKey := entry.Name() // YYYY-MM

		// Scan PDFs in this month directory
		pdfFiles, err := pm.scanMonthDirectory(monthDir, monthKey)
		if err != nil {
			log.Log(log.Error, "[PDFManager] Failed to scan month directory %s: %v", monthDir, err)
			continue
		}

		if len(pdfFiles) > 0 {
			newFiles[monthKey] = pdfFiles
		}
	}

	// Update the cached files
	pm.mu.Lock()
	pm.pdfFiles = newFiles
	pm.mu.Unlock()

	// Log summary
	totalFiles := 0
	for _, files := range newFiles {
		totalFiles += len(files)
	}
	log.Log(log.Info, "[PDFManager] Scan complete: found %d PDF files across %d months", totalFiles, len(newFiles))
}

// scanMonthDirectory scans a specific month directory for PDF files
func (pm *PDFManager) scanMonthDirectory(dirPath, monthKey string) ([]PDFInfo, error) {
	var pdfInfos []PDFInfo

	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	parts := strings.Split(monthKey, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid month key format: %s", monthKey)
	}
	year, month := parts[0], parts[1]

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".pdf") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			log.Log(log.Error, "[PDFManager] Failed to get file info for %s: %v", file.Name(), err)
			continue
		}

		pdfInfo := PDFInfo{
			Year:     year,
			Month:    month,
			FileName: file.Name(),
			FilePath: filepath.Join(dirPath, file.Name()),
			FileSize: info.Size(),
			ModTime:  info.ModTime().Format(time.RFC3339),
		}

		// Check if it's an overview file
		if matches := overviewPattern.FindStringSubmatch(file.Name()); matches != nil {
			pdfInfo.IsOverview = true
		} else if matches := pdfFilePattern.FindStringSubmatch(file.Name()); matches != nil {
			// Extract member name and convert underscores back to spaces for display
			memberName := strings.ReplaceAll(matches[3], "_", " ")
			pdfInfo.MemberName = memberName
		} else {
			// Skip files that don't match expected patterns
			log.Log(log.Debug, "[PDFManager] Skipping file with unexpected name: %s", file.Name())
			continue
		}

		pdfInfos = append(pdfInfos, pdfInfo)
	}

	// Sort by member name (overview files first)
	sort.Slice(pdfInfos, func(i, j int) bool {
		if pdfInfos[i].IsOverview != pdfInfos[j].IsOverview {
			return pdfInfos[i].IsOverview
		}
		return pdfInfos[i].MemberName < pdfInfos[j].MemberName
	})

	return pdfInfos, nil
}

// GetPDFList returns a list of available PDFs, optionally filtered
func (pm *PDFManager) GetPDFList(year, month, memberName string) []PDFInfo {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	var results []PDFInfo

	// If specific year/month requested
	if year != "" && month != "" {
		// Normalize month to 2-digit format
		monthInt := 0
		fmt.Sscanf(month, "%d", &monthInt)
		monthKey := fmt.Sprintf("%s-%02d", year, monthInt)
		if files, exists := pm.pdfFiles[monthKey]; exists {
			for _, pdf := range files {
				// Filter by member name if specified
				if memberName != "" {
					if !pdf.IsOverview && strings.EqualFold(pdf.MemberName, memberName) {
						results = append(results, pdf)
					}
				} else {
					results = append(results, pdf)
				}
			}
		}
	} else {
		// Return all PDFs
		for _, files := range pm.pdfFiles {
			for _, pdf := range files {
				// Filter by member name if specified
				if memberName != "" {
					if !pdf.IsOverview && strings.EqualFold(pdf.MemberName, memberName) {
						results = append(results, pdf)
					}
				} else {
					results = append(results, pdf)
				}
			}
		}
	}

	return results
}

// GetPDFFile returns the file path for a specific PDF
func (pm *PDFManager) GetPDFFile(year, month, memberName string, isOverview bool) (string, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	// Normalize month to 2-digit format
	monthInt := 0
	fmt.Sscanf(month, "%d", &monthInt)
	monthKey := fmt.Sprintf("%s-%02d", year, monthInt)
	files, exists := pm.pdfFiles[monthKey]
	if !exists {
		return "", fmt.Errorf("no PDFs found for %s", monthKey)
	}

	for _, pdf := range files {
		if isOverview && pdf.IsOverview {
			return pdf.FilePath, nil
		} else if !isOverview && !pdf.IsOverview && strings.EqualFold(pdf.MemberName, memberName) {
			return pdf.FilePath, nil
		}
	}

	if isOverview {
		return "", fmt.Errorf("overview PDF not found for %s", monthKey)
	}
	return "", fmt.Errorf("member PDF not found for %s in %s", memberName, monthKey)
}

// API Handlers

// handleListPDFs handles GET /api/billing/pdfs
func handleListPDFs(w http.ResponseWriter, r *http.Request) {
	if pdfManager == nil {
		writeError(w, http.StatusInternalServerError, "PDF manager not initialized")
		return
	}

	// Get query parameters
	year := r.URL.Query().Get("year")
	month := r.URL.Query().Get("month")
	memberName := r.URL.Query().Get("member")

	// Validate year if provided
	if year != "" && !validateYear(year) {
		writeError(w, http.StatusBadRequest, "Invalid year format")
		return
	}

	// Validate month if provided
	if month != "" && !validateMonth(month) {
		writeError(w, http.StatusBadRequest, "Invalid month format")
		return
	}

	// Get filtered PDF list
	pdfs := pdfManager.GetPDFList(year, month, memberName)

	// Group by month for better organization
	type MonthGroup struct {
		Year  string    `json:"year"`
		Month string    `json:"month"`
		PDFs  []PDFInfo `json:"pdfs"`
	}

	monthMap := make(map[string]*MonthGroup)
	for _, pdf := range pdfs {
		key := fmt.Sprintf("%s-%s", pdf.Year, pdf.Month)
		if _, exists := monthMap[key]; !exists {
			monthMap[key] = &MonthGroup{
				Year:  pdf.Year,
				Month: pdf.Month,
				PDFs:  []PDFInfo{},
			}
		}
		monthMap[key].PDFs = append(monthMap[key].PDFs, pdf)
	}

	// Convert to sorted slice
	var groups []MonthGroup
	for _, group := range monthMap {
		groups = append(groups, *group)
	}

	// Sort by year and month descending (newest first)
	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Year != groups[j].Year {
			return groups[i].Year > groups[j].Year
		}
		return groups[i].Month > groups[j].Month
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total": len(pdfs),
		"data":  groups,
	})
}

// handleDownloadPDF handles GET /api/billing/pdfs/download
func handleDownloadPDF(w http.ResponseWriter, r *http.Request) {
	if pdfManager == nil {
		writeError(w, http.StatusInternalServerError, "PDF manager not initialized")
		return
	}

	// Get query parameters
	year := r.URL.Query().Get("year")
	month := r.URL.Query().Get("month")
	memberName := r.URL.Query().Get("member")
	isOverview := r.URL.Query().Get("type") == "overview"

	// Validate required parameters
	if year == "" || month == "" {
		writeError(w, http.StatusBadRequest, "Year and month are required")
		return
	}

	if !isOverview && memberName == "" {
		writeError(w, http.StatusBadRequest, "Member name is required for non-overview PDFs")
		return
	}

	// Validate formats
	if !validateYear(year) {
		writeError(w, http.StatusBadRequest, "Invalid year format")
		return
	}

	if !validateMonth(month) {
		writeError(w, http.StatusBadRequest, "Invalid month format")
		return
	}

	// Get the PDF file path
	filePath, err := pdfManager.GetPDFFile(year, month, memberName, isOverview)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		log.Log(log.Error, "[API] Failed to open PDF file: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to open PDF file")
		return
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		log.Log(log.Error, "[API] Failed to get PDF file info: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to get file information")
		return
	}

	// Generate download filename
	var downloadName string
	// Normalize month to 2-digit format for filename
	monthInt := 0
	fmt.Sscanf(month, "%d", &monthInt)
	monthFormatted := fmt.Sprintf("%02d", monthInt)

	if isOverview {
		downloadName = fmt.Sprintf("%s_%s-Monthly_Overview.pdf", year, monthFormatted)
	} else {
		// Convert spaces to underscores in member name for filename
		safeMemberName := strings.ReplaceAll(memberName, " ", "_")
		downloadName = fmt.Sprintf("%s_%s-IBP-Service_%s.pdf", year, monthFormatted, safeMemberName)
	}

	// Set headers
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", downloadName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// Stream the file
	_, err = io.Copy(w, file)
	if err != nil {
		log.Log(log.Error, "[API] Failed to stream PDF file: %v", err)
	}
}

// Validation helpers
func validateYear(year string) bool {
	if len(year) != 4 {
		return false
	}
	for _, c := range year {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func validateMonth(month string) bool {
	// Accept both "01" and "1" formats
	if len(month) == 1 {
		month = "0" + month
	}
	if len(month) != 2 {
		return false
	}
	for _, c := range month {
		if c < '0' || c > '9' {
			return false
		}
	}
	// Check valid month range
	monthInt := int(month[0]-'0')*10 + int(month[1]-'0')
	return monthInt >= 1 && monthInt <= 12
}
