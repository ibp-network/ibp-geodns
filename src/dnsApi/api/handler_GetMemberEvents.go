package api

import (
	dat "ibp-geodns/src/common/data"
	"time"
)

// handle_GetMemberEvents returns downtime events for a member within a time range.
func handle_GetMemberEvents(req Request) Response {
	start, err := time.Parse(time.RFC3339, req.Parameters.StartTime)
	if err != nil {
		return Response{Result: "invalid startTime"}
	}
	end, err := time.Parse(time.RFC3339, req.Parameters.EndTime)
	if err != nil {
		return Response{Result: "invalid endTime"}
	}

	events, err := dat.GetMemberEvents(req.Parameters.MemberName, req.Parameters.Domain, start, end)
	if err != nil {
		return Response{Result: err.Error()}
	}

	return Response{Result: events}
}
