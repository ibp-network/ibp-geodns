package nats

import (
	"encoding/json"
	"strings"
	"time"

	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

// handleDnsUsageRequest responds to "dns.usage.getUsage" requests
func handleDnsUsageRequest(m *nats.Msg) {
	var req UsageRequest
	if err := json.Unmarshal(m.Data, &req); err != nil {
		log.Log(log.Error, "handleDnsUsageRequest: unmarshal error: %v", err)
		return
	}

	records, err := retrieveLocalUsageRecords(req.StartDate, req.EndDate, req.Domain, req.MemberName, req.Country)
	if err != nil {
		log.Log(log.Error, "retrieveLocalUsageRecords: %v", err)
		return
	}

	resp := UsageResponse{
		NodeID:       State.NodeID,
		UsageRecords: records,
	}
	dataBytes, _ := json.Marshal(resp)

	if m.Reply != "" {
		_ = PublishMsgWithReply(m.Reply, "", dataBytes)
	} else {
		_ = Publish("dns.usage.usageData", dataBytes)
	}
}

// retrieveLocalUsageRecords uses data usage functions from data/usage.go
func retrieveLocalUsageRecords(startDate, endDate, domain, member, country string) ([]UsageRecord, error) {
	var results []UsageRecord

	// parse startDate, endDate as "YYYY-MM-DD"
	sd := strings.TrimSpace(startDate)
	ed := strings.TrimSpace(endDate)
	if len(sd) != 10 || len(ed) != 10 {
		return nil, nil // or return an error
	}

	// We'll merge domain-level, member-level, or country-level usage from data/usage.go
	// data.GetUsageByDomain, data.GetUsageByMember, data.GetUsageByCountry
	//
	// We'll build a final list. For brevity, handle each scenario:

	if domain != "" && member != "" {
		// get usage by domain+member
		recs, err := dat.GetUsageByMember(domain, member, parseDate(sd), parseDate(ed))
		if err != nil {
			return nil, err
		}
		for _, r := range recs {
			if country == "" || strings.EqualFold(country, r.CountryCode) {
				res := UsageRecord{
					Date:        r.Date,
					Domain:      r.Domain,
					MemberName:  tryString(r.MemberName),
					CountryCode: r.CountryCode,
					Hits:        r.Hits,
				}
				results = append(results, res)
			}
		}
	} else if domain != "" {
		// usage by domain only
		recs, err := dat.GetUsageByDomain(domain, parseDate(sd), parseDate(ed))
		if err != nil {
			return nil, err
		}
		for _, r := range recs {
			if country == "" || strings.EqualFold(country, r.CountryCode) {
				res := UsageRecord{
					Date:        r.Date,
					Domain:      r.Domain,
					CountryCode: r.CountryCode,
					Hits:        r.Hits,
				}
				// memberName is unknown if usage_daily doesn't have that
				if r.MemberName.Valid {
					res.MemberName = r.MemberName.String
				}
				results = append(results, res)
			}
		}
	} else {
		// no domain => can do a global query by country or usage?
		// for simplicity, do getUsageByCountry, then filter.
		recs, err := dat.GetUsageByCountry(parseDate(sd), parseDate(ed))
		if err != nil {
			return nil, err
		}
		for _, r := range recs {
			if country == "" || strings.EqualFold(country, r.CountryCode) {
				res := UsageRecord{
					Date:        r.Date,
					CountryCode: r.CountryCode,
					Hits:        r.Hits,
				}
				results = append(results, res)
			}
		}
	}

	return results, nil
}

func parseDate(d string) time.Time {
	t, _ := time.Parse("2006-01-02", d)
	return t
}

// tryString handles sql.NullString -> string
func tryString(ns interface{}) string {
	if ns == nil {
		return ""
	}
	switch v := ns.(type) {
	case string:
		return v
	}
	return ""
}
