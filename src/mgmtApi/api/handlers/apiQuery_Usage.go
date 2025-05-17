package api

import (
	"net/http"
	"time"

	dat "ibp-geodns/src/common/data/mysql"
	types "ibp-geodns/src/mgmtApi/api/types"
)

// ApiQuery_Usage handles /api/usage requests
func ApiQuery_Usage(w http.ResponseWriter, r *http.Request, req types.Request) types.Response {
	switch req.Method {
	case "byClassc":
		return apiQuery_UsageByClassc(r)
	case "byCountry":
		return apiQuery_UsageByCountry(r)
	case "byMember":
		return apiQuery_UsageByMember(r)
	case "byDomain":
		return apiQuery_UsageByDomain(r)
	default:
		return types.Response{Error: "Invalid usage method"}
	}
}

// parseRange parses start and end date values from query parameters.
// Dates should be provided in YYYY-MM-DD format. If not provided, the
// range defaults to the last 30 days.
func parseRange(r *http.Request) (time.Time, time.Time, error) {
	q := r.URL.Query()
	endStr := q.Get("end")
	startStr := q.Get("start")

	var end time.Time
	var err error
	if endStr == "" {
		end = time.Now().UTC()
	} else {
		end, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}

	var start time.Time
	if startStr == "" {
		start = end.AddDate(0, 0, -30)
	} else {
		start, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
	}
	return start, end, nil
}

func apiQuery_UsageByCountry(r *http.Request) types.Response {
	start, end, err := parseRange(r)
	if err != nil {
		return types.Response{Error: err.Error()}
	}
	res, err := dat.GetUsageByCountry(start, end)
	if err != nil {
		return types.Response{Error: err.Error()}
	}
	return types.Response{Result: res}
}

func apiQuery_UsageByClassc(r *http.Request) types.Response {
	start, end, err := parseRange(r)
	if err != nil {
		return types.Response{Error: err.Error()}
	}
	res, err := dat.GetUsageByClassC(start, end)
	if err != nil {
		return types.Response{Error: err.Error()}
	}
	return types.Response{Result: res}
}

func apiQuery_UsageByMember(r *http.Request) types.Response {
	start, end, err := parseRange(r)
	if err != nil {
		return types.Response{Error: err.Error()}
	}
	res, err := dat.GetUsageByMember(start, end)
	if err != nil {
		return types.Response{Error: err.Error()}
	}
	return types.Response{Result: res}
}

func apiQuery_UsageByDomain(r *http.Request) types.Response {
	start, end, err := parseRange(r)
	if err != nil {
		return types.Response{Error: err.Error()}
	}
	res, err := dat.GetUsageByDomain(start, end)
	if err != nil {
		return types.Response{Error: err.Error()}
	}
	return types.Response{Result: res}
}
