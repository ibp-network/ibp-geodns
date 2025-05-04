package api

import (
	"net/http"

	"ibp-geodns/src/common/data"
)

// ApiQuery_Usage handles /api/usage requests
func ApiQuery_Usage(w http.ResponseWriter, r *http.Request, req ApiRequest) ApiResponse {
	switch req.Method {
	case "byClassc":
		return apiQuery_UsageByClassc()
	case "byCountry":
		return apiQuery_UsageByCountry()
	case "byMember":
		return apiQuery_UsageByMember()
	case "byDomain":
		return apiQuery_UsageByDomain()
	default:
		return ApiResponse{Error: "Invalid usage method"}
	}
}

// Example usage grouping
func apiQuery_UsageByCountry() ApiResponse {
	s := data.GetStats()
	// Return the raw stats or transform them as needed
	return ApiResponse{Result: s}
}

func apiQuery_UsageByClassc() ApiResponse {
	return ApiResponse{Result: "TODO usage byClassc not yet implemented"}
}

func apiQuery_UsageByMember() ApiResponse {
	return ApiResponse{Result: "TODO usage byMember not yet implemented"}
}

func apiQuery_UsageByDomain() ApiResponse {
	return ApiResponse{Result: "TODO usage byDomain not yet implemented"}
}
