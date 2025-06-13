package nats

/*   Collator‑specific runtime logic
     --------------------------------
     ‑ keeps proposals / votes in memory
     ‑ writes PASSed proposals to MySQL via data2
     ‑ aggregates DNS‑usage periodically
*/

import (
	"encoding/json"
	"sync"
	"time"

	data2 "ibp-geodns/src/common/data2"
	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

// ---------------------------------------------------------------------------
//  PUBLIC ENTRY POINT
// ---------------------------------------------------------------------------

// StartCollatorServices is invoked once by the IBPCollator binary *after*
// Connect() and EnableCollatorRole().
func StartCollatorServices() error {
	if _, err := Subscribe(State.SubjectVote, handleVote); err != nil {
		return err
	}
	if _, err := Subscribe(State.SubjectFinalize, handleFinalize); err != nil {
		return err
	}
	if _, err := Subscribe("dns.usage.usageData", handleUsageData); err != nil {
		return err
	}

	go startUsageCollector()
	go startMemoryJanitor()
	return nil
}

// ---------------------------------------------------------------------------
//  PROPOSAL / VOTE / FINALISE HANDLERS
// ---------------------------------------------------------------------------

var (
	voteMu sync.Mutex
	// proposalID -> set[nodeID]bool(agree)
	voteMap = make(map[string]map[string]bool)
)

// ---------------------------------------------------------------------------
//  DNS‑USAGE AGGREGATION
// ---------------------------------------------------------------------------

func handleUsageData(m *nats.Msg) {
	var resp UsageResponse
	if err := json.Unmarshal(m.Data, &resp); err != nil {
		log.Log(log.Error, "[collator] usage‑data unmarshal: %v", err)
		return
	}
	if len(resp.UsageRecords) == 0 {
		return
	}

	// Fan‑out to data2 (same logic the old collator had)
	records := make([]data2.UsageRecord, 0, len(resp.UsageRecords))
	for _, r := range resp.UsageRecords {
		dt, err := time.Parse("2006-01-02", r.Date)
		if err != nil {
			continue
		}
		records = append(records, data2.UsageRecord{
			Date:        dt,
			NodeID:      resp.NodeID,
			Domain:      r.Domain,
			MemberName:  r.MemberName,
			Asn:         r.Asn,
			NetworkName: r.NetworkName,
			CountryCode: r.CountryCode,
			CountryName: r.CountryName,
			IsIPv6:      false,
			Hits:        r.Hits,
		})
	}
	if err := data2.StoreUsageRecords(records); err != nil {
		log.Log(log.Error, "[collator] StoreUsageRecords: %v", err)
	}
}

// periodically query every DNS node for *today*’s usage slice
func startUsageCollector() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		today := time.Now().UTC().Format("2006-01-02")
		req := data2.UsageRequest{
			StartDate: today,
			EndDate:   today,
		}
		raw, err := RequestAllDnsUsage(req, 20*time.Second)
		if err != nil {
			log.Log(log.Error, "[collator] RequestAllDnsUsage: %v", err)
			continue
		}
		if len(raw) == 0 {
			continue
		}

		var recs []data2.UsageRecord
		for _, r := range raw {
			dt, err := time.Parse("2006-01-02", r.Date)
			if err != nil {
				continue
			}
			recs = append(recs, data2.UsageRecord{
				Date:        dt,
				NodeID:      State.NodeID,
				Domain:      r.Domain,
				MemberName:  r.MemberName,
				Asn:         r.Asn,
				NetworkName: r.NetworkName,
				CountryCode: r.CountryCode,
				CountryName: r.CountryName,
				IsIPv6:      false,
				Hits:        r.Hits,
			})
		}
		_ = data2.StoreUsageRecords(recs)
		log.Log(log.Info, "[collator] stored %d aggregated DNS‑usage record(s)", len(recs))
	}
}

// ---------------------------------------------------------------------------
//  MEMORY JANITOR (proposal expiry)
// ---------------------------------------------------------------------------

func startMemoryJanitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		data2.ExpireStaleProposals()
	}
}
