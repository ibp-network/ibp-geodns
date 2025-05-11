package api

import (
	"encoding/json"
	"net/http"
)

func router(w http.ResponseWriter, r *http.Request) {
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
		res = handle_Initialize(w, r, req)
	case "lookup":
		res = handle_Lookup(w, r, req)
	case "list":
		res = handle_List(w, r, req)
	case "getDomainInfo":
		res = handle_GetDomainInfo(w, r, req)
	case "getAllDomains":
		res = handle_GetAllDomains(w, r, req)
	case "getDomainKeys":
		res = handle_GetDomainKeys(w, r, req)
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
