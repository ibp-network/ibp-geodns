package api

import (
	"encoding/json"
	"net/http"
)

func dnsApiRouter(w http.ResponseWriter, r *http.Request) {
	var req Request
	var res Response

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		writeDnsResponse(w, Response{Result: "Invalid Request"})
		return
	}

	switch req.Method {
	case "initialize":
		res = handle_Init(req)
	case "lookup":
		res = handle_DNSQuery(req)
	case "list":
		res = handle_GetDomainList(req)
	case "getDomainInfo":
		res = handle_GetDomainInfo(req)
	case "getAllDomains":
		res = handle_GetAllDomains(req)
	case "getDomainKeys":
		res = handle_GetDomainKeys(req)
	default:
		writeDnsResponse(w, Response{Result: "Invalid Request"})
		return
	}

	writeDnsResponse(w, res)
}

func writeDnsResponse(w http.ResponseWriter, res Response) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}
