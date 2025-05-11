package api

import (
	"net/http"

	dat "ibp-geodns/src/common/data"
	types "ibp-geodns/src/mgmtApi/api/types"
)

// ApiQuery_Usage handles /api/usage requests
func ApiQuery_Usage(w http.ResponseWriter, r *http.Request, req types.Request) types.Response {
	switch req.Method {
	case "byClassc":
		return apiQuery_UsageByClassc()
	case "byCountry":
		return apiQuery_UsageByCountry()
	case "byMember":
		return apiQuery_UsageByMember()
	case "byDomain":
		return apiQuery_UsageByDomain()
	default:
		return types.Response{Error: "Invalid usage method"}
	}
}

// Example usage grouping
func apiQuery_UsageByCountry() types.Response {
	s := dat.GetStats()
	// Return the raw stats or transform them as needed
	return types.Response{Result: s}
}

func apiQuery_UsageByClassc() types.Response {
	return types.Response{Result: "TODO usage byClassc not yet implemented"}
}

func apiQuery_UsageByMember() types.Response {
	return types.Response{Result: "TODO usage byMember not yet implemented"}
}

func apiQuery_UsageByDomain() types.Response {
	return types.Response{Result: "TODO usage byDomain not yet implemented"}
}
