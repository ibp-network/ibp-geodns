package data2

import (
	"database/sql"
	"encoding/json"
	"time"

	"ibp-geodns/src/common/matrix"
)

// NetStatusRecord represents a single status transition event.
type NetStatusRecord struct {
	CheckType int
	CheckName string
	CheckURL  string
	Domain    string
	Member    string
	Status    bool
	IsIPv6    bool
	StartTime time.Time
	EndTime   sql.NullTime
	Error     string
	VoteData  map[string]bool
	Extra     map[string]interface{}
}

func InsertNetStatus(rec NetStatusRecord) error {
	jVotes, _ := json.Marshal(rec.VoteData)
	jExtra, _ := json.Marshal(rec.Extra)

	q := `INSERT INTO member_events
			(check_type,check_name,endpoint,domain_name,member_name,status,
			 is_ipv6,start_time,error,vote_data,additional_data)
		  VALUES (?,?,?,?,?,?,?,?,?,?,?)
		  ON DUPLICATE KEY UPDATE
			status       = VALUES(status),
			vote_data    = VALUES(vote_data),
			end_time     = IF(VALUES(status)=1,NOW(),NULL)`

	_, err := DB.Exec(q,
		rec.CheckType,
		rec.CheckName,
		rec.CheckURL,
		rec.Domain,
		rec.Member,
		boolToTiny(rec.Status),
		boolToTiny(rec.IsIPv6),
		rec.StartTime,
		nullOrString(rec.Error),
		string(jVotes),
		string(jExtra),
	)
	if err == nil && !rec.Status {
		// Only notify on NEW offline events (Status=false on insert/update)
		go matrix.NotifyMemberOffline(
			rec.Member,
			intToCheckType(rec.CheckType),
			rec.CheckName,
			rec.Domain,
			rec.CheckURL,
			rec.IsIPv6,
			rec.Error,
			rec.StartTime,
		)
	}
	return err
}

func CloseOpenEvent(rec NetStatusRecord) error {
	q := `UPDATE member_events
			SET end_time = NOW(), status = 1
		  WHERE check_type=? AND check_name=? AND endpoint=?
			AND domain_name=? AND member_name=? AND is_ipv6=?
			AND status=0 AND end_time IS NULL`
	_, err := DB.Exec(q,
		rec.CheckType,
		rec.CheckName,
		rec.CheckURL,
		rec.Domain,
		rec.Member,
		boolToTiny(rec.IsIPv6),
	)
	return err
}

func boolToTiny(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullOrString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func intToCheckType(t int) string {
	switch t {
	case 1:
		return "site"
	case 2:
		return "domain"
	case 3:
		return "endpoint"
	default:
		return "unknown"
	}
}
