package data2

import (
	"database/sql"
	"encoding/json"
	"ibp-geodns/src/common/matrix"
	"time"
)

// -----------------------------------------------------------------------------
// TYPES
// -----------------------------------------------------------------------------

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

// -----------------------------------------------------------------------------
// HELPERS
// -----------------------------------------------------------------------------

func ctToString(ct int) string {
	switch ct {
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

// -----------------------------------------------------------------------------
// DB OPERATIONS + MATRIX NOTIFICATIONS
// -----------------------------------------------------------------------------

func InsertNetStatus(rec NetStatusRecord) error {
	jVotes, _ := json.Marshal(rec.VoteData)
	jExtra, _ := json.Marshal(rec.Extra)

	// Ensure StartTime is UTC
	if rec.StartTime.Location() != time.UTC {
		rec.StartTime = rec.StartTime.UTC()
	}

	q := `INSERT INTO member_events
		(check_type,check_name,endpoint,domain_name,member_name,status,is_ipv6,start_time,error,vote_data,additional_data)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)
		ON DUPLICATE KEY UPDATE
		  status      = VALUES(status),
		  vote_data   = VALUES(vote_data),
		  end_time    = IF(VALUES(status)=1,UTC_TIMESTAMP(),NULL)`

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
		// New outage ⇒ alert
		matrix.NotifyMemberOffline(
			rec.Member,
			ctToString(rec.CheckType),
			rec.CheckName,
			rec.Domain,
			rec.CheckURL,
			rec.IsIPv6,
			rec.Error,
		)
	}

	return err
}

func CloseOpenEvent(rec NetStatusRecord) error {
	q := `UPDATE member_events
		SET end_time = UTC_TIMESTAMP(), status = 1
		WHERE check_type=? AND check_name=? AND endpoint=? AND domain_name=? AND member_name=? AND is_ipv6=? AND status=0 AND end_time IS NULL`

	_, err := DB.Exec(q,
		rec.CheckType,
		rec.CheckName,
		rec.CheckURL,
		rec.Domain,
		rec.Member,
		boolToTiny(rec.IsIPv6),
	)

	if err == nil {
		// Outage resolved ⇒ notify
		matrix.NotifyMemberOnline(
			rec.Member,
			ctToString(rec.CheckType),
			rec.CheckName,
			rec.Domain,
			rec.CheckURL,
			rec.IsIPv6,
		)
	}

	return err
}
