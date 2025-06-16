package mysql

import (
	"database/sql"
	"fmt"
	"time"
)

func DeleteEvent(eventID int64) error {
	query := `
		DELETE FROM member_events
		WHERE id = ?
	`
	_, err := DB.Exec(query, eventID)
	if err != nil {
		return fmt.Errorf("failed to delete event with ID %d: %w", eventID, err)
	}
	return nil
}

func InsertEvent(event EventRecord) (int64, error) {
	query := `
		INSERT INTO member_events
			(member_name, check_type, check_name, domain_name, endpoint, status, start_time, error, additional_data, is_ipv6)
		VALUES
			(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := DB.Exec(
		query,
		event.MemberName,
		event.CheckType,
		event.CheckName,
		event.DomainName,
		event.Endpoint,
		event.Status,
		event.StartTime,
		event.ErrorText,
		event.AdditionalData,
		event.IsIPv6,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert event: %w", err)
	}
	return result.LastInsertId()
}

func UpdateEventEndTime(eventID int64, endTime time.Time) error {
	query := `
		UPDATE member_events
		SET end_time = ?
		WHERE id = ?
	`
	_, err := DB.Exec(query, endTime, eventID)
	if err != nil {
		return fmt.Errorf("failed to update event end time: %w", err)
	}
	return nil
}

func FindOpenOfflineEvent(memberName, checkType, checkName, domainName, endpoint string, isIPv6 bool) (*EventRecord, error) {
	var row *sql.Row

	if checkType == "endpoint" {
		query := `
		SELECT id, member_name, check_type, check_name, domain_name, endpoint, status, start_time, end_time, error, additional_data, is_ipv6
		FROM member_events
		WHERE member_name = ? AND check_type = 'endpoint' AND check_name = ? AND domain_name = ? AND endpoint = ? AND status = FALSE AND end_time IS NULL AND is_ipv6 = ?
		`
		row = DB.QueryRow(query, memberName, checkName, domainName, endpoint, isIPv6)
	} else if checkType == "domain" {
		query := `
		SELECT id, member_name, check_type, check_name, domain_name, endpoint, status, start_time, end_time, error, additional_data, is_ipv6
		FROM member_events
		WHERE member_name = ? AND check_type = 'domain' AND check_name = ? AND domain_name = ? AND status = FALSE AND end_time IS NULL AND is_ipv6 = ?
		`
		row = DB.QueryRow(query, memberName, checkName, domainName, isIPv6)
	} else if checkType == "site" {
		query := `
		SELECT id, member_name, check_type, check_name, domain_name, endpoint, status, start_time, end_time, error, additional_data, is_ipv6
		FROM member_events
		WHERE member_name = ? AND check_type = 'site' AND check_name = ? AND status = FALSE AND end_time IS NULL AND is_ipv6 = ?
		`
		row = DB.QueryRow(query, memberName, checkName, isIPv6)
	}

	var event EventRecord
	err := row.Scan(
		&event.ID,
		&event.MemberName,
		&event.CheckType,
		&event.CheckName,
		&event.DomainName,
		&event.Endpoint,
		&event.Status,
		&event.StartTime,
		&event.EndTime,
		&event.ErrorText,
		&event.AdditionalData,
		&event.IsIPv6,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to find open offline event: %w", err)
	}
	return &event, nil
}

func GetEvents(memberName string, start, end time.Time) ([]EventRecord, error) {
	query := `
		SELECT id, member_name, check_type, check_name, domain_name, endpoint, status, start_time, end_time, error, additional_data, is_ipv6
		FROM member_events
		WHERE member_name = ? AND start_time >= ? AND start_time <= ?
	`
	rows, err := DB.Query(query, memberName, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var res []EventRecord
	for rows.Next() {
		var ev EventRecord
		err := rows.Scan(
			&ev.ID,
			&ev.MemberName,
			&ev.CheckType,
			&ev.CheckName,
			&ev.DomainName,
			&ev.Endpoint,
			&ev.Status,
			&ev.StartTime,
			&ev.EndTime,
			&ev.ErrorText,
			&ev.AdditionalData,
			&ev.IsIPv6,
		)
		if err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		res = append(res, ev)
	}
	return res, nil
}

func FetchEvents(memberName, domainName string, start, end time.Time) ([]EventRecord, error) {
	args := []interface{}{memberName, start, end}
	query := `
		SELECT id, member_name, check_type, check_name, domain_name, endpoint, status, start_time, end_time, error, additional_data, is_ipv6
		FROM member_events
		WHERE member_name = ? AND start_time >= ? AND start_time <= ?
	`

	if domainName != "" {
		query += " AND domain_name = ?"
		args = append(args, domainName)
	}
	query += " ORDER BY start_time"

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events: %w", err)
	}
	defer rows.Close()

	var events []EventRecord
	for rows.Next() {
		var e EventRecord
		if err := rows.Scan(
			&e.ID,
			&e.MemberName,
			&e.CheckType,
			&e.CheckName,
			&e.DomainName,
			&e.Endpoint,
			&e.Status,
			&e.StartTime,
			&e.EndTime,
			&e.ErrorText,
			&e.AdditionalData,
			&e.IsIPv6,
		); err != nil {
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}
		events = append(events, e)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return events, nil
}
