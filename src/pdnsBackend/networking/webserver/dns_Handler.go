package webserver

import (
	"encoding/json"
	"ibp-geodns/src/pdnsBackend/networking/webserver/dns"
	"net/http"
)

func handleDnsQuery(w http.ResponseWriter, r *http.Request) {
	var req dns.Request
	var res dns.Response

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		writeDnsResponse(w, dns.Response{Result: "Invalid Request"})
		return
	}

	switch req.Method {
	case "initialize":
		res = dns.DnsQuery_Initialize(w, r, req)
	case "lookup":
		res = dns.DnsQuery_Lookup(w, r, req)
	case "list":
		res = dns.DnsQuery_List(w, r, req)
	case "getDomainInfo":
		res = dns.DnsQuery_GetDomainInfo(w, r, req)
	case "getAllDomains":
		res = dns.DnsQuery_GetAllDomains(w, r, req)
	case "getDomainKeys":
		res = dns.DnsQuery_GetDomainKeys(w, r, req)
	default:
		writeDnsResponse(w, dns.Response{Result: "Invalid Request"})
		return
	}

	writeDnsResponse(w, res)
}

func writeDnsResponse(w http.ResponseWriter, res dns.Response) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(res)
	if err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
	}
}
