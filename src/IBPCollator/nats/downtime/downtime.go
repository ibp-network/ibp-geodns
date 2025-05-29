package downtime

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	log "ibp-geodns/src/common/logging"
	nconn "ibp-geodns/src/common/nats"

	"github.com/nats-io/nats.go"
)

// RequestAllDowntime sends a downtime request to "monitor.stats.getDowntime" with a unique reply subject,
// collects replies up to the specified timeout, and returns the aggregated events.
func RequestAllDowntime(startTime, endTime time.Time, memberName string, timeout time.Duration) ([]nconn.DowntimeEvent, error) {
	req := nconn.DowntimeRequest{
		StartTime:  startTime,
		EndTime:    endTime,
		MemberName: memberName,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("downtime request marshal error: %w", err)
	}

	inbox := fmt.Sprintf("%s.monitorStatsReply.%d", nconn.State.NodeID, time.Now().UnixNano())
	aggregated := make([]nconn.DowntimeEvent, 0, 16)

	// A channel on which we receive each response's events
	responseChan := make(chan []nconn.DowntimeEvent, 32)

	// Subscribe to the inbox; every message appends its events to responseChan
	sub, subErr := nconn.Subscribe(inbox, func(msg *nats.Msg) {
		var resp nconn.DowntimeResponse
		if unErr := json.Unmarshal(msg.Data, &resp); unErr != nil {
			log.Log(log.Error, "downtime aggregator: unmarshal error: %v", unErr)
			return
		}

		// Collect events
		responseChan <- resp.Events
	})
	if subErr != nil {
		return nil, fmt.Errorf("subscribe error: %w", subErr)
	}

	// Publish the request with a Reply set to our inbox
	err = nconn.PublishMsgWithReply("monitor.stats.getDowntime", inbox, data)
	if err != nil {
		sub.Unsubscribe()
		return nil, fmt.Errorf("publish downtime request error: %w", err)
	}

	// We'll wait for messages up to 'timeout' using a select-based approach
	timer := time.NewTimer(timeout)
	var mu sync.Mutex

	// Reader goroutine: we gather from the channel until we hit the timer
	doneChan := make(chan struct{})
	go func() {
		defer close(doneChan)
		for {
			select {
			case evts := <-responseChan:
				if evts == nil {
					// channel closed
					return
				}
				mu.Lock()
				aggregated = append(aggregated, evts...)
				mu.Unlock()
			case <-timer.C:
				// timed out
				return
			}
		}
	}()

	// Wait for the collection goroutine to complete after the timeout
	<-doneChan

	// Unsubscribe once done collecting
	sub.Unsubscribe()
	close(responseChan)

	mu.Lock()
	defer mu.Unlock()
	return aggregated, nil
}

// ParseTimeRange attempts to parse RFC3339 or YYYY-MM-DD date format.
func ParseTimeRange(startStr, endStr string) (time.Time, time.Time, error) {
	var startT, endT time.Time
	var err error

	startT, err = time.Parse(time.RFC3339, startStr)
	if err != nil {
		// fallback to date only
		startT, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("cannot parse start time: %v", err)
		}
	}

	endT, err = time.Parse(time.RFC3339, endStr)
	if err != nil {
		endT, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("cannot parse end time: %v", err)
		}
	}
	return startT, endT, nil
}
