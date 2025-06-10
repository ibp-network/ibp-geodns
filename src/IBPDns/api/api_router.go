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
		// CHANGED BELOW: return JSON with "result" field
		writeDnsErrorJSON(w, http.StatusBadRequest, "Bad request", err)
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
	case "getAllDomainMetadata":
		res = handle_GetAllDomainMetadata(req)
	case "getDomainKeys":
		res = handle_GetDomainKeys(req)
	case "getMemberEvents":
		res = handle_GetMemberEvents(req)
	default:
		log.Log(log.Warn, "dnsApiRouter: Unrecognized method: %s", req.Method)
		// We still have "result" in the JSON
		res = Response{Result: "Invalid Request"}
	}

	writeDnsResponse(w, res)
}

// writeDnsResponse writes a Response as JSON
func writeDnsResponse(w http.ResponseWriter, res Response) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		log.Log(log.Error, "dnsApiRouter: Error encoding JSON response: %v", err)
		// CHANGED BELOW: also produce JSON for this error
		writeDnsErrorJSON(w, http.StatusInternalServerError, "Error encoding response", err)
		return
	}
}

// CHANGED BELOW: A helper to return error JSON that includes "result" so PDNS doesn't crash
func writeDnsErrorJSON(w http.ResponseWriter, code int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	resp := map[string]interface{}{
		"result": message, // PDNS requires "result"
	}
	if err != nil {
		resp["error"] = err.Error()
	}
	_ = json.NewEncoder(w).Encode(resp)
}
