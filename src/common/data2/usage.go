package data2

import (
	"database/sql"

	log "ibp-geodns/src/common/logging"
)

/* ──────────────────────────────────────────────────────────────────────
   INSERT / UPSERT
   ────────────────────────────────────────────────────────────────────*/

// UpsertUsage writes or increments a row in ibpcollator_usage.
func UpsertUsage(r UsageRecord) error {
	q := `INSERT INTO ibpcollator_usage
	       (date,node_id,domain_name,member_name,network_asn,network_name,
	        country_code,country_name,is_ipv6,hits)
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

/* ──────────────────────────────────────────────────────────────────────
   BULK STORE
   ────────────────────────────────────────────────────────────────────*/

// StoreUsageRecords iterates over the slice and upserts every row.
func StoreUsageRecords(recs []UsageRecord) error {
	for _, r := range recs {
		if err := UpsertUsage(r); err != nil {
			log.Log(log.Error, "[data2] UpsertUsage err for %s/%s: %v",
				r.Domain, r.MemberName, err)
			return err
		}
	}
	return nil
}
