package api

import (
	"encoding/json"
	l "ibp-geodns/src/common/logging"
	"net/http"
)

func ApiQuery_Billing(w http.ResponseWriter, r *http.Request) ApiResponse {
	var req ApiRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		l.Log(l.Error, "Bad request: %v", http.StatusBadRequest)
		return ApiResponse{Result: ""}
	}

	var res ApiResponse

	switch req.Method {
	case "upcoming":
		res = apiQuery_BillingUpcoming(req)
	case "current":
		res = apiQuery_BillingCurrent(req)
	case "historical":
		res = apiQuery_BillingHistorical(req)
	default:
		res = ApiResponse{Error: "Invalid request"}
	}

	return res
}

func apiQuery_BillingUpcoming(params ApiRequest) ApiResponse {
	return ApiResponse{Result: ""}
}

func apiQuery_BillingCurrent(params ApiRequest) ApiResponse {
	return ApiResponse{Result: ""}
}

func apiQuery_BillingHistorical(params ApiRequest) ApiResponse {

	return ApiResponse{Result: ""}
}
