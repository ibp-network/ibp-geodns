package data2

import (
	"database/sql"
	log "ibp-geodns/src/common/logging"
	"ibp-geodns/src/common/nats"
	"time"
)

// UsageRecord mirrors ibpcollator_usage
type UsageRecord struct {
	Date        time.Time
	NodeID      string
	Domain      string
	MemberName  string
	Asn         string
	NetworkName string
	CountryCode string
	CountryName string
	IsIPv6      bool
	Hits        int
}

// UpsertUsage writes / increments a row in ibpcollator_usage.
func UpsertUsage(r UsageRecord) error {
	q := `INSERT INTO ibpcollator_usage
		(date,node_id,domain_name,member_name,network_asn,network_name,country_code,country_name,is_ipv6,hits)
		VALUES (?,?,?,?,?,?,?,?,?,?)
		ON DUPLICATE KEY UPDATE hits = hits + VALUES(hits)`

	ipFlag := "ipv4"
	if r.IsIPv6 {
		ipFlag = "ipv6"
	}

	_, err := DB.Exec(q,
		r.Date.Format("2006-01-02"),
		r.NodeID,
		r.Domain,
		nullOrEmpty(r.MemberName),
		nullOrEmpty(r.Asn),
		nullOrEmpty(r.NetworkName),
		nullOrEmpty(r.CountryCode),
		nullOrEmpty(r.CountryName),
		ipFlag,
		r.Hits,
	)
	return err
}

func nullOrEmpty(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// StoreUsageRecords persists a batch of NATS usage‑records.
func StoreUsageRecords(in []nats.UsageRecord) error {

	for _, rec := range in {
		// Parse the YYYY‑MM‑DD date coming from the DNS nodes
		dt, err := time.Parse("2006-01-02", rec.Date)
		if err != nil {
			log.Log(log.Warn,
				"[data2] StoreUsageRecords: invalid date %q – skipping (%v)",
				rec.Date, err)
			continue
		}

		u := UsageRecord{
			Date:        dt,
			NodeID:      nats.State.NodeID, // the collator that writes the row
			Domain:      rec.Domain,
			MemberName:  rec.MemberName,
			Asn:         rec.Asn,
			NetworkName: rec.NetworkName,
			CountryCode: rec.CountryCode,
			CountryName: rec.CountryName,
			IsIPv6:      false, // TBD when the wire‑format is upgraded
			Hits:        rec.Hits,
		}

		if err := UpsertUsage(u); err != nil {
			return err // bubble up so caller can log/handle
		}
	}
	return nil
}
