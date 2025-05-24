package api

import (
	"encoding/json"
	"net/http"

	log "ibp-geodns/src/common/logging"
)

// dnsApiRouter is the main entrypoint for PDNS remote backend requests.
func dnsApiRouter(w http.ResponseWriter, r *http.Request) {
	// Basic logging of the incoming request
	log.Log(log.Info, "dnsApiRouter: received HTTP %s from %s", r.Method, r.RemoteAddr)

	var req Request
	var res Response

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		log.Log(log.Warn, "dnsApiRouter: failed to parse JSON body: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	// Log the method we received (e.g. "lookup", "getDomainInfo", etc.)
	log.Log(log.Info, "dnsApiRouter: request.Method=%s, request.Parameters=%+v", req.Method, req.Parameters)

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
	case "getMemberEvents":
		res = handle_GetMemberEvents(req)
	default:
		log.Log(log.Warn, "dnsApiRouter: unknown method '%s'", req.Method)
		res = Response{Result: "Invalid Request"}
	}

	writeDnsResponse(w, res)
}

func writeDnsResponse(w http.ResponseWriter, res Response) {
	w.Header().Set("Content-Type", "application/json")
	encErr := json.NewEncoder(w).Encode(res)
	if encErr != nil {
		log.Log(log.Error, "dnsApiRouter: error encoding JSON response: %v", encErr)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}
