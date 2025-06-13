package mysql

import (
	"database/sql"
	"time"
)

var DB *sql.DB

type EventRecord struct {
	ID             int64
	MemberName     string
	CheckType      string
	CheckName      string
	DomainName     sql.NullString
	Endpoint       sql.NullString
	Status         bool
	StartTime      time.Time
	EndTime        sql.NullTime
	ErrorText      sql.NullString
	AdditionalData sql.NullString
	IsIPv6         bool
}
