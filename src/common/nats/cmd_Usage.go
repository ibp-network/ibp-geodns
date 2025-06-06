package nats

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

// handleDnsUsageRequest responds to "dns.usage.getUsage" requests
func handleDnsUsageRequest(m *nats.Msg) {
	log.Log(log.Debug,
		"[NATS] handleDnsUsageRequest: subject=%s reply=%s",
		m.Subject, m.Reply)

	var req UsageRequest
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Log(log.Error, "[NATS] handleDnsUsageRequest: unmarshal error: %v", err)
		return
	}

	log.Log(log.Debug,
		"[NATS] handleDnsUsageRequest: StartDate=%s EndDate=%s Domain=%s MemberName=%s Country=%s",
		req.StartDate, req.EndDate, req.Domain, req.MemberName, req.Country)

	records, err := retrieveLocalUsageRecords(req.StartDate, req.EndDate, req.Domain, req.MemberName, req.Country)
	if err != nil {
		log.Log(log.Error,
			"[NATS] handleDnsUsageRequest: retrieveLocalUsageRecords error: %v",
			err)
		return
	}

	resp := UsageResponse{
		NodeID:       State.NodeID,
		UsageRecords: records,
	}
	dataBytes, _ := json.Marshal(resp)

	if m.Reply != "" {
		// direct reply
		log.Log(log.Debug,
			"[NATS] handleDnsUsageRequest: replying to %s with %d usage records",
			m.Reply, len(records))
		_ = PublishMsgWithReply(m.Reply, "", dataBytes)
	} else {
		// fallback broadcast
		log.Log(log.Debug,
			"[NATS] handleDnsUsageRequest: publishing usageData with %d usage records",
			len(records))
		_ = Publish("dns.usage.usageData", dataBytes)
	}
}

func retrieveLocalUsageRecords(
	startDate, endDate, domain, member, country string,
) ([]UsageRecord, error) {
	log.Log(log.Debug,
		"[NATS] retrieveLocalUsageRecords: start=%s end=%s domain=%s member=%s country=%s",
		startDate, endDate, domain, member, country)

	sd := strings.TrimSpace(startDate)
	ed := strings.TrimSpace(endDate)
	if len(sd) != 10 || len(ed) != 10 {
		log.Log(log.Warn,
			"[NATS] retrieveLocalUsageRecords: invalid date format => start=%s end=%s",
			sd, ed)
		return nil, nil
	}

	sTime, _ := time.Parse("2006-01-02", sd)
	eTime, _ := time.Parse("2006-01-02", ed)

	var results []UsageRecord

	if domain != "" && member != "" {
		recs, err := dat.GetUsageByMember(domain, member, sTime, eTime)
		if err != nil {
			return nil, err
		}
		for _, r := range recs {
			if country == "" || strings.EqualFold(country, r.CountryCode) {
				results = append(results, UsageRecord{
					Date:        r.Date,
					Domain:      r.Domain,
					MemberName:  r.MemberName,
					CountryCode: r.CountryCode,
					Asn:         r.Asn,
					NetworkName: r.NetworkName,
					CountryName: r.CountryName,
					Hits:        r.Hits,
				})
			}
		}
	} else if domain != "" {
		recs, err := dat.GetUsageByDomain(domain, sTime, eTime)
		if err != nil {
			return nil, err
		}
		for _, r := range recs {
			if country == "" || strings.EqualFold(country, r.CountryCode) {
				results = append(results, UsageRecord{
					Date:        r.Date,
					Domain:      r.Domain,
					MemberName:  r.MemberName,
					CountryCode: r.CountryCode,
					Asn:         r.Asn,
					NetworkName: r.NetworkName,
					CountryName: r.CountryName,
					Hits:        r.Hits,
				})
			}
		}
	} else {
		// no domain => global query
		recs, err := dat.GetUsageByCountry(sTime, eTime)
		if err != nil {
			return nil, err
		}
		for _, r := range recs {
			if country == "" || strings.EqualFold(country, r.CountryCode) {
				results = append(results, UsageRecord{
					Date:        r.Date,
					Domain:      r.Domain,
					MemberName:  r.MemberName,
					CountryCode: r.CountryCode,
					Asn:         r.Asn,
					NetworkName: r.NetworkName,
					CountryName: r.CountryName,
					Hits:        r.Hits,
				})
			}
		}
	}

	log.Log(log.Debug,
		"[NATS] retrieveLocalUsageRecords: returning %d usage records",
		len(results))
	return results, nil
}

// RequestAllDnsUsage sends a usage request to "dns.usage.getUsage" with a unique inbox.
// It waits for all IBPDns nodes to reply or until timeout. Returns aggregated usage records.
func RequestAllDnsUsage(req UsageRequest, timeout time.Duration) ([]UsageRecord, error) {
	dnsCount := countNodesByRole("IBPDns")
	if dnsCount == 0 {
		return nil, fmt.Errorf("no IBPDns nodes found, cannot gather usage")
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("usage request marshal error: %w", err)
	}

	inbox := fmt.Sprintf("%s.usageReply.%d", State.NodeID, time.Now().UnixNano())
	responseChan := make(chan []UsageRecord, dnsCount)

	sub, subErr := Subscribe(inbox, func(msg *nats.Msg) {
		var resp UsageResponse
		if unErr := json.Unmarshal(msg.Data, &resp); unErr != nil {
			log.Log(log.Error, "[NATS] RequestAllDnsUsage: unmarshal error: %v", unErr)
			return
		}
		responseChan <- resp.UsageRecords
	})
	if subErr != nil {
		return nil, fmt.Errorf("subscribe error: %w", subErr)
	}

	err = PublishMsgWithReply("dns.usage.getUsage", inbox, data)
	if err != nil {
		sub.Unsubscribe()
		return nil, fmt.Errorf("publish usage request error: %w", err)
	}

	log.Log(log.Debug,
		"[NATS] RequestAllDnsUsage: expecting %d replies from IBPDns nodes",
		dnsCount)

	aggregated := make([]UsageRecord, 0, dnsCount*10)
	timer := time.NewTimer(timeout)
	var mu sync.Mutex

	done := make(chan struct{})
	go func() {
		defer close(done)
		received := 0
		for {
			select {
			case recs := <-responseChan:
				received++
				mu.Lock()
				aggregated = append(aggregated, recs...)
				mu.Unlock()
				if received >= dnsCount {
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
		"[NATS] RequestAllDnsUsage: done collecting => total usage records=%d",
		finalCount)

	return aggregated, nil
}

// handleDnsUsageData is invoked when we receive usage data from a node on "dns.usage.usageData"
func handleDnsUsageData(m *nats.Msg) {
	var resp UsageResponse
	if err := json.Unmarshal(m.Data, &resp); err != nil {
		log.Log(log.Error, "[NATS] handleDnsUsageData: unmarshal error: %v", err)
		return
	}
	log.Log(log.Debug, "[NATS] handleDnsUsageData: got %d usage records from node=%s",
		len(resp.UsageRecords), resp.NodeID)
}
