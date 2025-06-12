package data2

import (
	"database/sql"
	"encoding/json"
	"time"
)

// NetStatusRecord maps to ibpcollator_netStatus
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
	VoteData  map[string]bool // NodeID -> agree
	Extra     map[string]interface{}
}

func InsertNetStatus(rec NetStatusRecord) error {
	jVotes, _ := json.Marshal(rec.VoteData)
	jExtra, _ := json.Marshal(rec.Extra)

	q := `INSERT INTO ibpcollator_netStatus
		(check_type,check_name,check_url,domain_name,member_name,status,is_ipv6,start_time,error,vote_data,additional_data)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)
		ON DUPLICATE KEY UPDATE status = VALUES(status), vote_data = VALUES(vote_data),
			end_time = IF(VALUES(status)=1,NOW(),NULL)`

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
	return err
}

// CloseOpenEvent sets end_time when a member comes back online.
func CloseOpenEvent(rec NetStatusRecord) error {
	q := `UPDATE ibpcollator_netStatus SET end_time = NOW(), status = 1
		WHERE check_type=? AND check_name=? AND check_url=? AND domain_name=? AND member_name=? AND is_ipv6=? AND status=0 AND end_time IS NULL`
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
