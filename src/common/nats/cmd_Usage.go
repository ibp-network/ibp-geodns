package nats

import (
	"encoding/json"

	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

// handleDnsUsageRequest listens for requests on "dns.usage.getUsage"
func handleDnsUsageRequest(m *nats.Msg) {
	var req UsageRequest
	err := json.Unmarshal(m.Data, &req)
	if err != nil {
		log.Log(log.Error, "handleDnsUsageRequest: unmarshal error: %v", err)
		return
	}

	usage := retrieveLocalUsageRecords(req.StartDate, req.EndDate, req.Domain, req.MemberName, req.Country)

	resp := UsageResponse{
		NodeID:       State.NodeID,
		UsageRecords: usage,
	}
	data, _ := json.Marshal(resp)

	// If the request is a direct request, reply
	if m.Reply != "" {
		err = Publish(m.Reply, data)
		if err != nil {
			log.Log(log.Error, "Failed to publish usage response: %v", err)
		}
		return
	}

	// Otherwise, publish to "dns.usage.usageData"
	_ = Publish("dns.usage.usageData", data)
}

// retrieveLocalUsageRecords fetches usage from local stats or DB.
func retrieveLocalUsageRecords(startDate, endDate, domain, member, country string) []UsageRecord {
	// Implementation depends on your usage DB
	// Return slice of usage records
	return []UsageRecord{}
}
