package dnsApi

import (
	"encoding/json"
	dnsApi "ibp-geodns/src/dnsApi/api/dnsHandler"
	"net/http"
)

func handleDnsQuery(w http.ResponseWriter, r *http.Request) {
	var req dnsApi.Request
	var res dnsApi.Response

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		writeDnsResponse(w, dnsApi.Response{Result: "Invalid Request"})
		return
	}

	switch req.Method {
	case "initialize":
		res = dnsApi.DnsQuery_Initialize(w, r, req)
	case "lookup":
		res = dnsApi.DnsQuery_Lookup(w, r, req)
	case "list":
		res = dnsApi.DnsQuery_List(w, r, req)
	case "getDomainInfo":
		res = dnsApi.DnsQuery_GetDomainInfo(w, r, req)
	case "getAllDomains":
		res = dnsApi.DnsQuery_GetAllDomains(w, r, req)
	case "getDomainKeys":
		res = dnsApi.DnsQuery_GetDomainKeys(w, r, req)
	default:
		writeDnsResponse(w, dnsApi.Response{Result: "Invalid Request"})
		return
	}

	writeDnsResponse(w, res)
}

func writeDnsResponse(w http.ResponseWriter, res dnsApi.Response) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}
