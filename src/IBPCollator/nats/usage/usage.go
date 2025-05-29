package usage

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	log "ibp-geodns/src/common/logging"
	nconn "ibp-geodns/src/common/nats"

	"github.com/nats-io/nats.go"
)

// RequestAllUsage sends a usage request to "dns.usage.getUsage" with a unique reply subject,
// collects replies up to the specified timeout, and returns the aggregated usage records.
func RequestAllUsage(startDate, endDate, domain, member, country string, timeout time.Duration) ([]nconn.UsageRecord, error) {
	req := nconn.UsageRequest{
		StartDate:  startDate,
		EndDate:    endDate,
		Domain:     domain,
		MemberName: member,
		Country:    country,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("usage request marshal error: %w", err)
	}

	// Create a unique inbox for replies
	inbox := fmt.Sprintf("%s.dnsUsageReply.%d", nconn.State.NodeID, time.Now().UnixNano())
	aggregated := make([]nconn.UsageRecord, 0, 16)

	// Channel for each response's usage records
	responseChan := make(chan []nconn.UsageRecord, 32)

	sub, subErr := nconn.Subscribe(inbox, func(msg *nats.Msg) {
		var resp nconn.UsageResponse
		if unErr := json.Unmarshal(msg.Data, &resp); unErr != nil {
			log.Log(log.Error, "usage aggregator: unmarshal error: %v", unErr)
			return
		}
		responseChan <- resp.UsageRecords
	})
	if subErr != nil {
		return nil, fmt.Errorf("subscribe error: %w", subErr)
	}

	// Publish the request with "Reply" set to the inbox
	err = nconn.PublishMsgWithReply("dns.usage.getUsage", inbox, data)
	if err != nil {
		sub.Unsubscribe()
		return nil, fmt.Errorf("publish request error: %w", err)
	}

	timer := time.NewTimer(timeout)
	var mu sync.Mutex

	doneChan := make(chan struct{})
	go func() {
		defer close(doneChan)
		for {
			select {
			case recs := <-responseChan:
				if recs == nil {
					return
				}
				mu.Lock()
				aggregated = append(aggregated, recs...)
				mu.Unlock()
			case <-timer.C:
				return
			}
		}
	}()

	<-doneChan
	sub.Unsubscribe()
	close(responseChan)

	mu.Lock()
	defer mu.Unlock()
	return aggregated, nil
}
