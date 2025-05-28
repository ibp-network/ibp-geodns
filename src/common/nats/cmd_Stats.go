package nats

import (
	"encoding/json"
	"time"

	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

// handleMonitorStatsRequest listens for requests on "monitor.stats.getDowntime".
func handleMonitorStatsRequest(m *nats.Msg) {
	var req DowntimeRequest
	err := json.Unmarshal(m.Data, &req)
	if err != nil {
		log.Log(log.Error, "handleMonitorStatsRequest: unmarshal error: %v", err)
		return
	}

	// Query local downtime events in date range
	// The logic below is an integration point to your data storage
	events := retrieveLocalDowntimeEvents(req.StartTime, req.EndTime, req.MemberName)

	resp := DowntimeResponse{
		NodeID: State.NodeID,
		Events: events,
	}
	data, _ := json.Marshal(resp)

	// If the request has a Reply subject (inbox), respond directly
	if m.Reply != "" {
		err = Publish(m.Reply, data)
		if err != nil {
			log.Log(log.Error, "Failed to publish downtime response: %v", err)
		}
		return
	}

	// Otherwise, publish to a well-known subject, e.g. "monitor.stats.downtimeData"
	_ = Publish("monitor.stats.downtimeData", data)
}

// retrieveLocalDowntimeEvents is your local data retrieval logic.
func retrieveLocalDowntimeEvents(start, end time.Time, memberName string) []DowntimeEvent {
	// Implementation depends on your MySQL or in-memory DB code
	// Returns an array of DowntimeEvent
	var results []DowntimeEvent
	// ...
	return results
}
