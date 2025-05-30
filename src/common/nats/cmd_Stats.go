package nats

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

// handleMonitorStatsRequest responds to downtime requests: "monitor.stats.getDowntime"
func handleMonitorStatsRequest(m *nats.Msg) {
	log.Log(log.Debug, "[NATS] handleMonitorStatsRequest: subject=%s reply=%s", m.Subject, m.Reply)

	var req DowntimeRequest
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Log(log.Error, "[NATS] handleMonitorStatsRequest: unmarshal error: %v", err)
		return
	}

	log.Log(log.Debug, "[NATS] handleMonitorStatsRequest: StartTime=%v EndTime=%v MemberName=%s",
		req.StartTime, req.EndTime, req.MemberName)

	// retrieve local downtime events
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

	// If the request provided an m.Reply subject, we respond directly
	if m.Reply != "" {
		log.Log(log.Debug,
			"[NATS] handleMonitorStatsRequest: replying to %s with %d events",
			m.Reply, len(events))
		_ = PublishMsgWithReply(m.Reply, "", dataBytes)

	} else {
		// Otherwise, we can broadcast to "monitor.stats.downtimeData"
		log.Log(log.Debug,
			"[NATS] handleMonitorStatsRequest: publishing downtimeData with %d events",
			len(events))
		_ = Publish("monitor.stats.downtimeData", dataBytes)
	}
}

func retrieveLocalDowntimeEvents(memberName string, start, end time.Time) ([]DowntimeEvent, error) {
	log.Log(log.Debug,
		"[NATS] retrieveLocalDowntimeEvents: memberName=%s start=%v end=%v",
		memberName, start, end)

	rawEvents, err := dat.GetMemberEvents(memberName, "", start, end)
	if err != nil {
		log.Log(log.Error,
			"[NATS] retrieveLocalDowntimeEvents: data.GetMemberEvents error: %v",
			err)
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
	log.Log(log.Debug,
		"[NATS] retrieveLocalDowntimeEvents: returning %d events",
		len(results))

	return results, nil
}

// RequestAllMonitorsDowntime sends a DowntimeRequest to "monitor.stats.getDowntime"
// with a unique inbox. Then we wait for all IBPMonitor nodes to reply or until timeout.
func RequestAllMonitorsDowntime(req DowntimeRequest, timeout time.Duration) ([]DowntimeEvent, error) {
	monitorCount := countNodesByRole("IBPMonitor")
	if monitorCount == 0 {
		return nil, fmt.Errorf("no IBPMonitor nodes found, cannot gather downtime")
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("downtime request marshal error: %w", err)
	}

	// We'll create a unique inbox subject where we expect replies
	inbox := fmt.Sprintf("%s.downtimeReply.%d", State.NodeID, time.Now().UnixNano())
	responseChan := make(chan []DowntimeEvent, monitorCount)

	// Subscribe to that inbox
	sub, subErr := Subscribe(inbox, func(msg *nats.Msg) {
		var resp DowntimeResponse
		if unErr := json.Unmarshal(msg.Data, &resp); unErr != nil {
			log.Log(log.Error, "[NATS] RequestAllMonitorsDowntime: unmarshal error: %v", unErr)
			return
		}
		responseChan <- resp.Events
	})
	if subErr != nil {
		return nil, fmt.Errorf("subscribe error: %w", subErr)
	}

	// Publish the request to "monitor.stats.getDowntime", with our inbox as the reply subject
	err = PublishMsgWithReply("monitor.stats.getDowntime", inbox, data)
	if err != nil {
		sub.Unsubscribe()
		return nil, fmt.Errorf("publish downtime request error: %w", err)
	}

	log.Log(log.Debug,
		"[NATS] RequestAllMonitorsDowntime: expecting %d replies from IBPMonitor nodes",
		monitorCount)

	aggregated := make([]DowntimeEvent, 0, monitorCount*10)
	timer := time.NewTimer(timeout)
	var mu sync.Mutex

	done := make(chan struct{})
	go func() {
		defer close(done)
		received := 0
		for {
			select {
			case evts := <-responseChan:
				received++
				mu.Lock()
				aggregated = append(aggregated, evts...)
				mu.Unlock()
				if received >= monitorCount {
					return
				}
			case <-timer.C:
				return
			}
		}
	}()

	<-done
	sub.Unsubscribe()
	close(responseChan)

	mu.Lock()
	finalCount := len(aggregated)
	mu.Unlock()

	log.Log(log.Debug,
		"[NATS] RequestAllMonitorsDowntime: done collecting => total events=%d",
		finalCount)

	return aggregated, nil
}
