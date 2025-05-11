package api

import (
	"net/http"

	log "ibp-geodns/src/common/logging"
	types "ibp-geodns/src/mgmtApi/api/types"
)

// ApiQuery_Billing processes incoming requests for /api/billing
func ApiQuery_Billing(w http.ResponseWriter, r *http.Request, req types.Request) types.Response {
	switch req.Method {
	case "upcoming":
		return apiQuery_BillingUpcoming(req)
	case "current":
		return apiQuery_BillingCurrent(req)
	case "historical":
		return apiQuery_BillingHistorical(req)
	default:
		log.Log(log.Warn, "Billing request with unknown method: %s", req.Method)
		return types.Response{Error: "Invalid billing method"}
	}
}

// Sub-handlers for each method:
func apiQuery_BillingUpcoming(params types.Request) types.Response {
	// Fill in any logic or DB lookups as needed
	return types.Response{Result: "Billing upcoming placeholder"}
}

func apiQuery_BillingCurrent(params types.Request) types.Response {
	return types.Response{Result: "Billing current placeholder"}
}

func apiQuery_BillingHistorical(params types.Request) types.Response {
	return types.Response{Result: "Billing historical placeholder"}
}
