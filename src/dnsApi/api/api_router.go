package api

import (
	"encoding/json"
	log "ibp-geodns/src/common/logging"
	"net/http"
)

// dnsApiRouter handles all JSON POST requests from PowerDNS or from curl
func dnsApiRouter(w http.ResponseWriter, r *http.Request) {
	// Just for debugging, show that we got a request
	log.Log(log.Debug, "dnsApiRouter: Received HTTP %s from %s", r.Method, r.RemoteAddr)

	// Try decoding the JSON
	var req Request
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		log.Log(log.Warn, "dnsApiRouter: JSON decode error: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	log.Log(log.Debug, "dnsApiRouter: req.Method=%s, req.Parameters=%+v", req.Method, req.Parameters)

	// Dispatch by req.Method
	var res Response
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
		log.Log(log.Warn, "dnsApiRouter: Unrecognized method: %s", req.Method)
		res = Response{Result: "Invalid Request"}
	}

	writeDnsResponse(w, res)
}

func writeDnsResponse(w http.ResponseWriter, res Response) {
	// Encode the response as JSON
	w.Header().Set("Content-Type", "application/json")

	// For debugging, show what we’re returning
	// (WARNING: can be verbose, but helps debugging)
	// log.Log(log.Debug, "dnsApiRouter: writing JSON response: %+v", res)

	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		log.Log(log.Error, "dnsApiRouter: Error encoding JSON response: %v", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}
