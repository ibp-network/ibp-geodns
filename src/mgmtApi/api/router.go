package api

import (
	"encoding/json"
	handlers "ibp-geodns/src/mgmtApi/api/handlers"
	types "ibp-geodns/src/mgmtApi/api/types"
	"net/http"
)

func router(w http.ResponseWriter, r *http.Request) {
	var req types.Request
	var res types.Response

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		writeDnsResponse(w, types.Response{Result: "Invalid Request"})
		return
	}

	switch req.Method {
	case "initialize":
		res = handlers.ApiQuery_Billing(w, r, req)
	case "lookup":
		res = handlers.ApiQuery_Member(w, r, req)
	case "list":
		res = handlers.ApiQuery_Status(w, r, req)
	case "getDomainInfo":
		res = handlers.ApiQuery_Usage(w, r, req)
	default:
		writeDnsResponse(w, types.Response{Result: "Invalid Request"})
		return
	}

	writeDnsResponse(w, res)
}

func writeDnsResponse(w http.ResponseWriter, res types.Response) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}
