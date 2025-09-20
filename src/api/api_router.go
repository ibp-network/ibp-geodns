package api

import (
	"encoding/json"
	"net/http"

	log "github.com/ibp-network/ibp-geodns-libs/logging"
)

func dnsApiRouter(w http.ResponseWriter, r *http.Request) {
	log.Log(log.Debug, "dnsApiRouter: Received HTTP %s from %s", r.Method, r.RemoteAddr)

	var req Request
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		log.Log(log.Warn, "dnsApiRouter: JSON decode error: %v", err)
		writeDnsErrorJSON(w, http.StatusBadRequest, "Bad request", err)
		return
	}

	log.Log(log.Debug, "dnsApiRouter: req.Method=%s, req.Parameters=%+v", req.Method, req.Parameters)

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
		res = Response{Result: "Invalid Request"}
	}

	writeDnsResponse(w, res)
}

func writeDnsResponse(w http.ResponseWriter, res Response) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		log.Log(log.Error, "dnsApiRouter: Error encoding JSON response: %v", err)
		writeDnsErrorJSON(w, http.StatusInternalServerError, "Error encoding response", err)
		return
	}
}

func writeDnsErrorJSON(w http.ResponseWriter, code int, message string, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	resp := map[string]interface{}{
		"result": message,
	}
	if err != nil {
		resp["error"] = err.Error()
	}
	_ = json.NewEncoder(w).Encode(resp)
}
