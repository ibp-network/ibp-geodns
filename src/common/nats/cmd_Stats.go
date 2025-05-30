package nats

import (
	"encoding/json"
	"time"

	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

// handleMonitorStatsRequest responds to downtime requests: "monitor.stats.getDowntime"
func handleMonitorStatsRequest(m *nats.Msg) {
	log.Log(log.Debug, "[NATS] handleMonitorStatsRequest: received request on subject=%s from reply=%s", m.Subject, m.Reply)

	var req DowntimeRequest
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Log(log.Error, "[NATS] handleMonitorStatsRequest: unmarshal error: %v", err)
		return
	}

	log.Log(log.Debug, "[NATS] handleMonitorStatsRequest: StartTime=%v EndTime=%v MemberName=%s",
		req.StartTime, req.EndTime, req.MemberName)

	events, err := retrieveLocalDowntimeEvents(req.MemberName, req.StartTime, req.EndTime)
	if err != nil {
		log.Log(log.Error, "[NATS] handleMonitorStatsRequest: error retrieving local downtime: %v", err)
		return
	}

	resp := DowntimeResponse{
		NodeID: State.NodeID,
		Events: events,
	}
	dataBytes, _ := json.Marshal(resp)

	if m.Reply != "" {
		log.Log(log.Debug, "[NATS] handleMonitorStatsRequest: replying to %s with %d events", m.Reply, len(events))
		_ = PublishMsgWithReply(m.Reply, "", dataBytes)
	} else {
		log.Log(log.Debug, "[NATS] handleMonitorStatsRequest: publishing downtimeData with %d events", len(events))
		_ = Publish("monitor.stats.downtimeData", dataBytes)
	}
}

// retrieveLocalDowntimeEvents uses data.GetMemberEvents from data/events.go
func retrieveLocalDowntimeEvents(memberName string, start, end time.Time) ([]DowntimeEvent, error) {
	log.Log(log.Debug, "[NATS] retrieveLocalDowntimeEvents: memberName=%s start=%v end=%v", memberName, start, end)
	rawEvents, err := dat.GetMemberEvents(memberName, "", start, end)
	if err != nil {
		log.Log(log.Error, "[NATS] retrieveLocalDowntimeEvents: error from data.GetMemberEvents: %v", err)
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

	log.Log(log.Debug, "[NATS] retrieveLocalDowntimeEvents: returning %d events", len(results))
	return results, nil
}
