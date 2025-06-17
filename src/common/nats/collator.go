package nats

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	data2 "ibp-geodns/src/common/data2"
	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

/*
 * collator.go – services that run only on IBPCollator nodes
 *
 * Key change:
 *   • StartUsageCollector now runs **hourly** (top of every UTC hour).
 *   • After we receive fresh totals we simply *overwrite* the previous
 *     value in MySQL (UpsertUsage has been made idempotent), so there
 *     is no risk of compounding counts.
 */

func parseDateFlexible(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}

	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse("2006-01-02 15:04:05", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, fmt.Errorf("unrecognised date format: %q", s)
}

func handleUsageData(m *nats.Msg) {
	var resp UsageResponse
	if err := json.Unmarshal(m.Data, &resp); err != nil {
		log.Log(log.Error, "[collator] usageData unmarshal: %v", err)
		return
	}
	if len(resp.UsageRecords) == 0 {
		return
	}

	records := make([]data2.UsageRecord, 0, len(resp.UsageRecords))
	for _, r := range resp.UsageRecords {
		dt, err := parseDateFlexible(r.Date)
		if err != nil {
			log.Log(log.Warn, "[collator] skipping record with invalid date %q: %v", r.Date, err)
			continue
		}
		records = append(records, data2.UsageRecord{
			Date:        dt,
			NodeID:      resp.NodeID, // keep origin node
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

	if len(records) == 0 {
		log.Log(log.Warn, "[collator] no valid usage records to store from node %s", resp.NodeID)
		return
	}

	if err := data2.StoreUsageRecords(records); err != nil {
		log.Log(log.Error, "[collator] StoreUsageRecords: %v", err)
	}
}

/* ----------------------------- HOURLY PULLER ------------------------------ */

func StartUsageCollector() {
	// Wait until the next top‑of‑hour, then run every hour.
	now := time.Now().UTC()
	next := now.Truncate(time.Hour).Add(time.Hour)
	time.Sleep(time.Until(next))

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		collectOnce()
		<-ticker.C
	}
}

func collectOnce() {
	period := time.Now().UTC().Format("2006-01-02")
	req := data2.UsageRequest{
		StartDate: period,
		EndDate:   period,
	}

	raw, err := RequestAllDnsUsage(req, 20*time.Second)
	if err != nil {
		log.Log(log.Error, "[collator] RequestAllDnsUsage: %v", err)
		return
	}
	if len(raw) == 0 {
		log.Log(log.Info, "[collator] no usage data returned from DNS nodes")
		return
	}

	records := make([]data2.UsageRecord, 0, len(raw))
	for _, r := range raw {
		dt, err := parseDateFlexible(r.Date)
		if err != nil {
			log.Log(log.Warn, "[collator] skipping aggregated record with invalid date %q: %v", r.Date, err)
			continue
		}
		records = append(records, data2.UsageRecord{
			Date:        dt,
			NodeID:      r.NodeID, // preserve DNS node‑id to keep rows unique
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

	if len(records) == 0 {
		log.Log(log.Warn, "[collator] all usage records were skipped due to bad dates")
		return
	}

	if err := data2.StoreUsageRecords(records); err != nil {
		log.Log(log.Error, "[collator] StoreUsageRecords: %v", err)
		return
	}
	log.Log(log.Info, "[collator] stored %d DNS‑usage record(s) for %s", len(records), period)
}

/* -------------------------- JANITOR REMAINS SAME -------------------------- */

func StartMemoryJanitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		data2.ExpireStaleProposals()
	}
}

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

	go StartUsageCollector()
	go StartMemoryJanitor()

	return nil
}
