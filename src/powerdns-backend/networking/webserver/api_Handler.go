package webserver

import (
	"encoding/json"
	"ibp-geodns/src/powerdns-backend/network/webserver/api"
	"net/http"
	"strings"
)

func handleApiQuery(w http.ResponseWriter, r *http.Request) {
	var req api.ApiRequest
	var res api.ApiResponse

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		writeApiResponse(w, api.ApiResponse{Error: "Bad request"})
		return
	}

	path := strings.ToLower(strings.Trim(r.URL.Path, "/"))

	switch path {
	case "api/billing":
		res = api.ApiQuery_Billing(w, r)
	case "api/member":
		res = api.ApiQuery_Member(w, r)
	case "api/status":
		res = api.ApiQuery_Status(w, r)
	case "api/usage":
		res = api.ApiQuery_Usage(w, r)
	default:
		writeApiResponse(w, api.ApiResponse{Error: "Invalid request"})
		return
	}

	writeApiResponse(w, res)
}

func writeApiResponse(w http.ResponseWriter, res api.ApiResponse) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}
