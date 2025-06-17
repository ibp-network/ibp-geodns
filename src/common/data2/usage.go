package data2

import (
	"database/sql"
	"fmt"
	"strings"

	log "ibp-geodns/src/common/logging"
)

/*
 * UpsertUsage persists **per‑node** usage totals coming from IBPDns
 * or from the collator’s own hourly aggregation.
 *
 *  • Primary key = date, node_id, domain_name, member_name,
 *                  network_asn, network_name, country_code, country_name, is_ipv6
 *
 *  • The row’s `hits` column is **replaced** with the latest total, NOT
 *    incremented.  This guarantees that importing the *same* period
 *    more than once is idempotent and does **not** compound data.
 */
func UpsertUsage(r UsageRecord) error {
	q := `INSERT INTO requests
	       (date, node_id, domain_name, member_name, network_asn, network_name,
	        country_code, country_name, is_ipv6, hits)
	       VALUES (?,?,?,?,?,?,?,?,?,?)
	       ON DUPLICATE KEY UPDATE
	         hits = VALUES(hits)`

	ipFlag := 0
	if r.IsIPv6 {
		ipFlag = 1
	}

	_, err := DB.Exec(
		q,
		r.Date.Format("2006-01-02"),
		r.NodeID,
		nullOrEmpty(r.Domain),
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

func StoreUsageRecords(recs []UsageRecord) error {
	var errs []string
	for _, r := range recs {
		if err := UpsertUsage(r); err != nil {
			log.Log(
				log.Error,
				"[data2] UpsertUsage error for domain=%s member=%s: %v",
				r.Domain, r.MemberName, err,
			)
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("StoreUsageRecords completed with %d error(s): %s",
			len(errs), strings.Join(errs, "; "))
	}
	return nil
}
