package api

import (
	"time"

	dat "github.com/ibp-network/ibp-geodns-libs/data"
	log "github.com/ibp-network/ibp-geodns-libs/logging"
)

func handle_GetMemberEvents(req Request) Response {
	p := req.Parameters
	log.Log(log.Debug, "handle_GetMemberEvents: memberName=%s, domain=%s, startTime=%s, endTime=%s",
		p.MemberName, p.Domain, p.StartTime, p.EndTime)

	start, err := time.Parse(time.RFC3339, p.StartTime)
	if err != nil {
		log.Log(log.Warn, "handle_GetMemberEvents: invalid startTime=%s", p.StartTime)
		return Response{Result: "invalid startTime"}
	}
	end, err := time.Parse(time.RFC3339, p.EndTime)
	if err != nil {
		log.Log(log.Warn, "handle_GetMemberEvents: invalid endTime=%s", p.EndTime)
		return Response{Result: "invalid endTime"}
	}

	events, err := dat.GetMemberEvents(p.MemberName, p.Domain, start, end)
	if err != nil {
		log.Log(log.Error, "handle_GetMemberEvents: error retrieving events: %v", err)
		return Response{Result: err.Error()}
	}

	log.Log(log.Debug, "handle_GetMemberEvents: returning %d events", len(events))
	return Response{Result: events}
}
