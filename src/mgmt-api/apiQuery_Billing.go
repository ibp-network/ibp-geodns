package mgmtApi

import (
	"net/http"

	l "ibp-geodns/src/common/logging"
)

// ApiQuery_Billing processes incoming requests for /api/billing
func ApiQuery_Billing(w http.ResponseWriter, r *http.Request, req ApiRequest) ApiResponse {
	switch req.Method {
	case "upcoming":
		return apiQuery_BillingUpcoming(req)
	case "current":
		return apiQuery_BillingCurrent(req)
	case "historical":
		return apiQuery_BillingHistorical(req)
	default:
		l.Log(l.Warn, "Billing request with unknown method: %s", req.Method)
		return ApiResponse{Error: "Invalid billing method"}
	}
}

// Sub-handlers for each method:
func apiQuery_BillingUpcoming(params ApiRequest) ApiResponse {
	// Fill in any logic or DB lookups as needed
	return ApiResponse{Result: "Billing upcoming placeholder"}
}

func apiQuery_BillingCurrent(params ApiRequest) ApiResponse {
	return ApiResponse{Result: "Billing current placeholder"}
}

func apiQuery_BillingHistorical(params ApiRequest) ApiResponse {
	return ApiResponse{Result: "Billing historical placeholder"}
}
