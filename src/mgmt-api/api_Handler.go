package mgmtApi

import (
	"encoding/json"
	"net/http"
	"strings"

	l "ibp-geodns/src/common/logging"
)

// HandleApiQuery is the single handler for all mgmt endpoints,
// invoked by main.go at /api/billing, /api/member, /api/status, /api/usage, etc.
func HandleApiQuery(w http.ResponseWriter, r *http.Request) {
	var req ApiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		writeApiResponse(w, ApiResponse{Error: "Bad request"})
		return
	}

	path := strings.ToLower(strings.Trim(r.URL.Path, "/"))
	var res ApiResponse

	switch path {
	case "api/billing":
		res = ApiQuery_Billing(w, r, req)
	case "api/member":
		res = ApiQuery_Member(w, r, req)
	case "api/status":
		res = ApiQuery_Status(w, r, req)
	case "api/usage":
		res = ApiQuery_Usage(w, r, req)
	default:
		res = ApiResponse{Error: "Invalid management endpoint"}
	}

	writeApiResponse(w, res)
}

// writeApiResponse writes the final JSON response to the caller.
func writeApiResponse(w http.ResponseWriter, res ApiResponse) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(res); err != nil {
		l.Log(l.Error, "Error encoding JSON response: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}

// Init performs any initialization for the mgmt API (no-op here).
func Init() {
	// If you have some initialization logic, do it here.
}
