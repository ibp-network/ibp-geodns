// ===== cmd_Stats.go =====
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
	log.Log(log.Debug, "[NATS] handleMonitorStatsRequest: subject=%s reply=%s from=%s",
		m.Subject, m.Reply, m.Header.Get("sender"))

	var req DowntimeRequest
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Log(log.Error, "[NATS] handleMonitorStatsRequest: unmarshal error: %v", err)
		// Send error response if we have a reply subject
		if m.Reply != "" {
			errResp := DowntimeResponse{
				NodeID: State.NodeID,
				Events: []DowntimeEvent{},
				Error:  fmt.Sprintf("unmarshal error: %v", err),
			}
			if data, err := json.Marshal(errResp); err == nil {
				PublishMsgWithReply(m.Reply, "", data)
			}
		}
		return
	}

	log.Log(log.Debug, "[NATS] handleMonitorStatsRequest: StartTime=%v EndTime=%v MemberName=%s",
		req.StartTime, req.EndTime, req.MemberName)

	// Validate request
	if req.EndTime.Before(req.StartTime) {
		log.Log(log.Error, "[NATS] handleMonitorStatsRequest: EndTime before StartTime")
		if m.Reply != "" {
			errResp := DowntimeResponse{
				NodeID: State.NodeID,
				Events: []DowntimeEvent{},
				Error:  "EndTime must be after StartTime",
			}
			if data, err := json.Marshal(errResp); err == nil {
				PublishMsgWithReply(m.Reply, "", data)
			}
		}
		return
	}

	// retrieve local downtime events
	events, err := retrieveLocalDowntimeEvents(req.MemberName, req.StartTime, req.EndTime)
	if err != nil {
		log.Log(log.Error, "[NATS] handleMonitorStatsRequest: error retrieving local downtime: %v", err)
		events = []DowntimeEvent{} // Send empty list on error
	}

	resp := DowntimeResponse{
		NodeID: State.NodeID,
		Events: events,
	}
	dataBytes, _ := json.Marshal(resp)

	if m.Reply != "" {
		log.Log(log.Debug,
			"[NATS] handleMonitorStatsRequest: replying to %s with %d events",
			m.Reply, len(events))
		_ = PublishMsgWithReply(m.Reply, "", dataBytes)
	} else {
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
		return nil, err
	}

	results := make([]DowntimeEvent, 0, len(rawEvents))
	for _, e := range rawEvents {
		// Only include offline events (status=false)
		if !e.Status {
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
				IsIPv6:     e.IsIPv6,
			})
		}
	}

	log.Log(log.Debug,
		"[NATS] retrieveLocalDowntimeEvents: found %d total events, returning %d downtime events for member=%s",
		len(rawEvents), len(results), memberName)

	return results, nil
}

// RequestAllMonitorsDowntime sends a DowntimeRequest to "monitor.stats.getDowntime"
func RequestAllMonitorsDowntime(req DowntimeRequest, timeout time.Duration) ([]DowntimeEvent, error) {
	monitorCount := countActiveMonitors()
	if monitorCount == 0 {
		return nil, fmt.Errorf("no active IBPMonitor nodes found")
	}

	log.Log(log.Debug, "[NATS] RequestAllMonitorsDowntime: requesting from %d active monitors", monitorCount)

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("downtime request marshal error: %w", err)
	}

	// Create unique inbox
	inbox := fmt.Sprintf("_INBOX.%s.downtimeReply.%d", State.NodeID, time.Now().UnixNano())

	// Use a map to track responses and avoid duplicates
	responseMap := make(map[string][]DowntimeEvent)
	var mu sync.Mutex

	// Subscribe to inbox
	sub, err := Subscribe(inbox, func(msg *nats.Msg) {
		var resp DowntimeResponse
		if err := json.Unmarshal(msg.Data, &resp); err != nil {
			log.Log(log.Error, "[NATS] RequestAllMonitorsDowntime: unmarshal error: %v", err)
			return
		}

		mu.Lock()
		if _, exists := responseMap[resp.NodeID]; !exists {
			responseMap[resp.NodeID] = resp.Events
			log.Log(log.Debug, "[NATS] RequestAllMonitorsDowntime: received %d events from %s",
				len(resp.Events), resp.NodeID)
		} else {
			log.Log(log.Warn, "[NATS] RequestAllMonitorsDowntime: duplicate response from %s ignored", resp.NodeID)
		}
		mu.Unlock()
	})
	if err != nil {
		return nil, fmt.Errorf("subscribe error: %w", err)
	}
	defer sub.Unsubscribe()

	// Publish request
	err = PublishMsgWithReply("monitor.stats.getDowntime", inbox, data)
	if err != nil {
		return nil, fmt.Errorf("publish downtime request error: %w", err)
	}

	// Wait for responses or timeout
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	checkInterval := 100 * time.Millisecond
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			// Timeout reached
			mu.Lock()
			receivedCount := len(responseMap)
			mu.Unlock()
			log.Log(log.Warn,
				"[NATS] RequestAllMonitorsDowntime: timeout after receiving %d/%d responses",
				receivedCount, monitorCount)
			goto done

		case <-ticker.C:
			// Check if we have all responses
			mu.Lock()
			if len(responseMap) >= monitorCount {
				mu.Unlock()
				log.Log(log.Debug, "[NATS] RequestAllMonitorsDowntime: received all %d responses", monitorCount)
				goto done
			}
			mu.Unlock()
		}
	}

done:
	// Aggregate all events
	mu.Lock()
	defer mu.Unlock()

	aggregated := make([]DowntimeEvent, 0)
	for nodeID, events := range responseMap {
		log.Log(log.Debug, "[NATS] RequestAllMonitorsDowntime: aggregating %d events from %s",
			len(events), nodeID)
		aggregated = append(aggregated, events...)
	}

	log.Log(log.Debug,
		"[NATS] RequestAllMonitorsDowntime: completed with %d total events from %d nodes",
		len(aggregated), len(responseMap))

	return aggregated, nil
}

// handleMonitorStatsData is invoked when we receive downtime data from "monitor.stats.downtimeData"
func handleMonitorStatsData(m *nats.Msg) {
	var resp DowntimeResponse
	if err := json.Unmarshal(m.Data, &resp); err != nil {
		log.Log(log.Error, "[NATS] handleMonitorStatsData: unmarshal error: %v", err)
		return
	}
	log.Log(log.Debug, "[NATS] handleMonitorStatsData: got %d downtime events from node=%s",
		len(resp.Events), resp.NodeID)
	// Collator would process these events here
}
