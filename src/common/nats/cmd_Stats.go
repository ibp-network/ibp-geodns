package nats

import (
	"encoding/json"
	"time"

	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"

	dat "ibp-geodns/src/common/data"
)

// handleMonitorStatsRequest responds to downtime requests: "monitor.stats.getDowntime"
func handleMonitorStatsRequest(m *nats.Msg) {
	var req DowntimeRequest
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Log(log.Error, "handleMonitorStatsRequest: unmarshal error: %v", err)
		return
	}

	// Query local downtime events from data/events.go
	start := req.StartTime
	end := req.EndTime
	member := req.MemberName

	// data.GetMemberEvents is from data/events.go or data usage
	events, err := retrieveLocalDowntimeEvents(member, start, end)
	if err != nil {
		log.Log(log.Error, "Error retrieving local downtime: %v", err)
		return
	}

	resp := DowntimeResponse{
		NodeID: State.NodeID,
		Events: events,
	}
	dataBytes, _ := json.Marshal(resp)

	if m.Reply != "" {
		_ = PublishMsgWithReply(m.Reply, "", dataBytes)
	} else {
		_ = Publish("monitor.stats.downtimeData", dataBytes)
	}
}

// retrieveLocalDowntimeEvents uses data.GetMemberEvents from data/events.go
func retrieveLocalDowntimeEvents(memberName string, start, end time.Time) ([]DowntimeEvent, error) {
	rawEvents, err := dat.GetMemberEvents(memberName, "", start, end)
	if err != nil {
		return nil, err
	}

	results := make([]DowntimeEvent, 0, len(rawEvents))
	for _, e := range rawEvents {
		results = append(results, DowntimeEvent{
			MemberName: e.MemberName,
			CheckType:  e.CheckType,
			CheckName:  e.CheckName,
			DomainName: e.DomainName,
			Endpoint:   e.Endpoint,
			Status:     e.Status,
			StartTime:  e.StartTime,
			EndTime:    e.EndTime,
			ErrorText:  e.ErrorText,
			Data:       e.Data,
		})
	}
	return results, nil
}
