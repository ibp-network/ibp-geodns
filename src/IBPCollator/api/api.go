package api

import (
	"encoding/json"
	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	"net/http"
	"time"
)

var (
	mux *http.ServeMux
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

	// Billing endpoints
	mux.HandleFunc("/api/billing/breakdown", corsMiddleware(handleBillingBreakdown))
	mux.HandleFunc("/api/billing/summary", corsMiddleware(handleBillingSummary))

	// Health check
	mux.HandleFunc("/api/health", corsMiddleware(handleHealth))

	addr := c.Local.CollatorApi.ListenAddress
	port := c.Local.CollatorApi.ListenPort

	log.Log(log.Info, "[CollatorAPI] Starting API server on %s:%s", addr, port)

	go func() {
		if err := http.ListenAndServe(addr+":"+port, mux); err != nil {
			log.Log(log.Fatal, "[CollatorAPI] Failed to start server: %v", err)
		}
	}()
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
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   cfg.GetVersion(),
	})
}
