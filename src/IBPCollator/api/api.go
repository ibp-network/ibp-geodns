package api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
)

var (
	mux         *http.ServeMux
	tlsConfig   *tls.Config
	tlsMutex    sync.RWMutex
	certPath    string
	keyPath     string
	lastCertMod time.Time
	lastKeyMod  time.Time
)

// PDF Management
type PDFInfo struct {
	Year       string `json:"year"`
	Month      string `json:"month"`
	MemberName string `json:"member_name,omitempty"`
	IsOverview bool   `json:"is_overview"`
	FileName   string `json:"file_name"`
	FilePath   string `json:"-"` // Don't expose full path in API
	FileSize   int64  `json:"file_size"`
	ModTime    string `json:"modified_time"`
}

type PDFManager struct {
	mu       sync.RWMutex
	pdfFiles map[string][]PDFInfo // key: "YYYY-MM"
	baseDir  string
}

var (
	pdfManager      *PDFManager
	pdfFilePattern  = regexp.MustCompile(`^(\d{4})_(\d{2})-IBP-Service_(.+)\.pdf$`)
	overviewPattern = regexp.MustCompile(`^(\d{4})_(\d{2})-Monthly_Overview\.pdf$`)
	monthDirPattern = regexp.MustCompile(`^\d{4}-\d{2}$`)
)

// CORS middleware
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "3600")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func Init() {
	log.Log(log.Info, "[CollatorAPI] Initializing API...")

	c := cfg.GetConfig()
	mux = http.NewServeMux()

	// Initialize PDF manager
	initPDFManager()

	// Request statistics endpoints
	mux.HandleFunc("/api/requests/country", corsMiddleware(handleRequestsByCountry))
	mux.HandleFunc("/api/requests/asn", corsMiddleware(handleRequestsByASN))
	mux.HandleFunc("/api/requests/service", corsMiddleware(handleRequestsByService))
	mux.HandleFunc("/api/requests/member", corsMiddleware(handleRequestsByMember))
	mux.HandleFunc("/api/requests/summary", corsMiddleware(handleRequestsSummary))

	// Downtime endpoints
	mux.HandleFunc("/api/downtime/events", corsMiddleware(handleDowntimeEvents))
	mux.HandleFunc("/api/downtime/current", corsMiddleware(handleCurrentDowntime))
	mux.HandleFunc("/api/downtime/summary", corsMiddleware(handleDowntimeSummary))

	// Member endpoints
	mux.HandleFunc("/api/members", corsMiddleware(handleMembers))
	mux.HandleFunc("/api/members/stats", corsMiddleware(handleMemberStats))

	// Service endpoints (NEW)
	mux.HandleFunc("/api/services", corsMiddleware(handleServices))
	mux.HandleFunc("/api/services/summary", corsMiddleware(handleServicesSummary))

	// Billing endpoints
	mux.HandleFunc("/api/billing/breakdown", corsMiddleware(handleBillingBreakdown))
	mux.HandleFunc("/api/billing/summary", corsMiddleware(handleBillingSummary))

	// PDF endpoints
	mux.HandleFunc("/api/billing/pdfs", corsMiddleware(handleListPDFs))
	mux.HandleFunc("/api/billing/pdfs/download", corsMiddleware(handleDownloadPDF))

	// Health check
	mux.HandleFunc("/api/health", corsMiddleware(handleHealth))

	addr := c.Local.CollatorApi.ListenAddress
	port := c.Local.CollatorApi.ListenPort

	// Check if SSL environment variables are set
	certPath = os.Getenv("SSL_CERT")
	keyPath = os.Getenv("SSL_KEY")

	if certPath != "" && keyPath != "" {
		// Initialize TLS configuration
		if err := loadTLSConfig(); err != nil {
			log.Log(log.Fatal, "[CollatorAPI] Failed to load TLS configuration: %v", err)
			return
		}

		// Start certificate watcher
		go watchCertificates()

		// Create HTTPS server
		server := &http.Server{
			Addr:    addr + ":" + port,
			Handler: mux,
			TLSConfig: &tls.Config{
				GetCertificate: getCertificate,
			},
		}

		log.Log(log.Info, "[CollatorAPI] Starting HTTPS API server on %s:%s", addr, port)
		go func() {
			if err := server.ListenAndServeTLS("", ""); err != nil {
				log.Log(log.Fatal, "[CollatorAPI] Failed to start HTTPS server: %v", err)
			}
		}()
	} else {
		// Start HTTP server (no SSL)
		log.Log(log.Info, "[CollatorAPI] Starting HTTP API server on %s:%s (no SSL configured)", addr, port)
		go func() {
			if err := http.ListenAndServe(addr+":"+port, mux); err != nil {
				log.Log(log.Fatal, "[CollatorAPI] Failed to start HTTP server: %v", err)
			}
		}()
	}
}

// loadTLSConfig loads the certificate and key from disk
func loadTLSConfig() error {
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return fmt.Errorf("failed to load certificate: %w", err)
	}

	tlsMutex.Lock()
	tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
	tlsMutex.Unlock()

	// Update modification times
	if certInfo, err := os.Stat(certPath); err == nil {
		lastCertMod = certInfo.ModTime()
	}
	if keyInfo, err := os.Stat(keyPath); err == nil {
		lastKeyMod = keyInfo.ModTime()
	}

	log.Log(log.Info, "[CollatorAPI] TLS configuration loaded successfully")
	return nil
}

// getCertificate is called by the TLS handshake to get the current certificate
func getCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	tlsMutex.RLock()
	defer tlsMutex.RUnlock()

	if tlsConfig != nil && len(tlsConfig.Certificates) > 0 {
		return &tlsConfig.Certificates[0], nil
	}
	return nil, fmt.Errorf("no certificate available")
}

// watchCertificates monitors certificate files for changes
func watchCertificates() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for range ticker.C {
		reloadNeeded := false

		// Check certificate file
		if certInfo, err := os.Stat(certPath); err == nil {
			if !certInfo.ModTime().Equal(lastCertMod) {
				reloadNeeded = true
				log.Log(log.Info, "[CollatorAPI] Certificate file changed, reloading...")
			}
		} else {
			log.Log(log.Error, "[CollatorAPI] Failed to stat certificate file: %v", err)
			continue
		}

		// Check key file
		if keyInfo, err := os.Stat(keyPath); err == nil {
			if !keyInfo.ModTime().Equal(lastKeyMod) {
				reloadNeeded = true
				log.Log(log.Info, "[CollatorAPI] Key file changed, reloading...")
			}
		} else {
			log.Log(log.Error, "[CollatorAPI] Failed to stat key file: %v", err)
			continue
		}

		// Reload if needed
		if reloadNeeded {
			if err := loadTLSConfig(); err != nil {
				log.Log(log.Error, "[CollatorAPI] Failed to reload TLS configuration: %v", err)
			} else {
				log.Log(log.Info, "[CollatorAPI] TLS configuration reloaded successfully")
			}
		}
	}
}

// Helper functions
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Log(log.Error, "[CollatorAPI] Failed to encode JSON: %v", err)
	}
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}

func parseTimeParams(r *http.Request) (time.Time, time.Time, error) {
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	// Default to today if not specified
	if startStr == "" {
		startStr = time.Now().UTC().Format("2006-01-02")
	}
	if endStr == "" {
		endStr = time.Now().UTC().Format("2006-01-02")
	}

	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	// End of day for end date
	end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	return start, end, nil
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	// Check SSL status
	sslEnabled := certPath != "" && keyPath != ""
	sslStatus := "disabled"
	if sslEnabled {
		tlsMutex.RLock()
		if tlsConfig != nil && len(tlsConfig.Certificates) > 0 {
			sslStatus = "enabled"
		} else {
			sslStatus = "error"
		}
		tlsMutex.RUnlock()
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   cfg.GetVersion(),
		"ssl":       sslStatus,
	})
}
