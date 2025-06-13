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

func handleDnsUsageRequest(m *nats.Msg) {
	log.Log(log.Debug,
		"[NATS] handleDnsUsageRequest: subject=%s reply=%s",
		m.Subject, m.Reply)

	var req UsageRequest
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Log(log.Error, "[NATS] handleDnsUsageRequest: unmarshal error: %v", err)
		if m.Reply != "" {
			errResp := UsageResponse{
				NodeID:       State.NodeID,
				UsageRecords: []UsageRecord{},
				Error:        fmt.Sprintf("unmarshal error: %v", err),
			}
			if data, err := json.Marshal(errResp); err == nil {
				PublishMsgWithReply(m.Reply, "", data)
			}
		}
		return
	}

	log.Log(log.Debug,
		"[NATS] handleDnsUsageRequest: StartDate=%s EndDate=%s Domain=%s MemberName=%s Country=%s",
		req.StartDate, req.EndDate, req.Domain, req.MemberName, req.Country)

	if req.StartDate > req.EndDate {
		log.Log(log.Error, "[NATS] handleDnsUsageRequest: StartDate after EndDate")
		if m.Reply != "" {
			errResp := UsageResponse{
				NodeID:       State.NodeID,
				UsageRecords: []UsageRecord{},
				Error:        "StartDate must be before or equal to EndDate",
			}
			if data, err := json.Marshal(errResp); err == nil {
				PublishMsgWithReply(m.Reply, "", data)
			}
		}
		return
	}

	records, err := retrieveLocalUsageRecords(req.StartDate, req.EndDate, req.Domain, req.MemberName, req.Country)
	if err != nil {
		log.Log(log.Error,
			"[NATS] handleDnsUsageRequest: retrieveLocalUsageRecords error: %v",
			err)
		records = []UsageRecord{}
	}

	resp := UsageResponse{
		NodeID:       State.NodeID,
		UsageRecords: records,
	}
	dataBytes, _ := json.Marshal(resp)

	if m.Reply != "" {
		log.Log(log.Debug,
			"[NATS] handleDnsUsageRequest: replying to %s with %d usage records",
			m.Reply, len(records))
		_ = PublishMsgWithReply(m.Reply, "", dataBytes)
	} else {
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
		return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD")
	}

	sTime, err := time.Parse("2006-01-02", sd)
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %w", err)
	}
	eTime, err := time.Parse("2006-01-02", ed)
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %w", err)
	}

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
		recs, err := dat.GetUsageByCountry(sTime, eTime)
		if err != nil {
			return nil, err
		}
		for _, r := range recs {
			if member != "" && r.MemberName != member {
				continue
			}
			if country != "" && !strings.EqualFold(country, r.CountryCode) {
				continue
			}
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

	log.Log(log.Debug,
		"[NATS] retrieveLocalUsageRecords: returning %d usage records",
		len(results))
	return results, nil
}

func RequestAllDnsUsage(req UsageRequest, timeout time.Duration) ([]UsageRecord, error) {
	dnsCount := countActiveDns()
	if dnsCount == 0 {
		return nil, fmt.Errorf("no active IBPDns nodes found")
	}

	log.Log(log.Debug, "[NATS] RequestAllDnsUsage: requesting from %d active DNS nodes", dnsCount)

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("usage request marshal error: %w", err)
	}

	inbox := fmt.Sprintf("_INBOX.%s.usageReply.%d", State.NodeID, time.Now().UnixNano())
	responseMap := make(map[string][]UsageRecord)
	var mu sync.Mutex

	sub, err := Subscribe(inbox, func(msg *nats.Msg) {
		var resp UsageResponse
		if err := json.Unmarshal(msg.Data, &resp); err != nil {
			log.Log(log.Error, "[NATS] RequestAllDnsUsage: unmarshal error: %v", err)
			return
		}

		mu.Lock()
		if _, exists := responseMap[resp.NodeID]; !exists {
			responseMap[resp.NodeID] = resp.UsageRecords
			log.Log(log.Debug, "[NATS] RequestAllDnsUsage: received %d records from %s",
				len(resp.UsageRecords), resp.NodeID)
		} else {
			log.Log(log.Warn, "[NATS] RequestAllDnsUsage: duplicate response from %s ignored", resp.NodeID)
		}
		mu.Unlock()
	})
	if err != nil {
		return nil, fmt.Errorf("subscribe error: %w", err)
	}
	defer sub.Unsubscribe()

	err = PublishMsgWithReply("dns.usage.getUsage", inbox, data)
	if err != nil {
		return nil, fmt.Errorf("publish usage request error: %w", err)
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	checkInterval := 100 * time.Millisecond
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timer.C:
			mu.Lock()
			receivedCount := len(responseMap)
			mu.Unlock()
			log.Log(log.Warn,
				"[NATS] RequestAllDnsUsage: timeout after receiving %d/%d responses",
				receivedCount, dnsCount)
			goto done

		case <-ticker.C:
			mu.Lock()
			if len(responseMap) >= dnsCount {
				mu.Unlock()
				log.Log(log.Debug, "[NATS] RequestAllDnsUsage: received all %d responses", dnsCount)
				goto done
			}
			mu.Unlock()
		}
	}

done:
	mu.Lock()
	defer mu.Unlock()

	aggregateMap := make(map[string]UsageRecord)

	for nodeID, records := range responseMap {
		log.Log(log.Debug, "[NATS] RequestAllDnsUsage: aggregating %d records from %s",
			len(records), nodeID)
		for _, rec := range records {
			key := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s",
				rec.Date, rec.Domain, rec.MemberName, rec.CountryCode,
				rec.Asn, rec.NetworkName, rec.CountryName)

			if existing, found := aggregateMap[key]; found {
				existing.Hits += rec.Hits
				aggregateMap[key] = existing
			} else {
				aggregateMap[key] = rec
			}
		}
	}

	aggregated := make([]UsageRecord, 0, len(aggregateMap))
	for _, rec := range aggregateMap {
		aggregated = append(aggregated, rec)
	}

	log.Log(log.Debug,
		"[NATS] RequestAllDnsUsage: completed with %d unique records from %d nodes",
		len(aggregated), len(responseMap))

	return aggregated, nil
}

func handleDnsUsageData(m *nats.Msg) {
	var resp UsageResponse
	if err := json.Unmarshal(m.Data, &resp); err != nil {
		log.Log(log.Error, "[NATS] handleDnsUsageData: unmarshal error: %v", err)
		return
	}
	markNodeHeard(resp.NodeID)

	log.Log(log.Debug, "[NATS] handleDnsUsageData: got %d usage records from node=%s",
		len(resp.UsageRecords), resp.NodeID)
}
