package mysql

import (
	"database/sql"
	"fmt"
	"time"
)

// DeleteEvent deletes an event from the database by its ID.
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

// InsertEvent inserts a new offline event into the database.
func InsertEvent(event EventRecord) (int64, error) {
	query := `
		INSERT INTO member_events 
		(member_name, check_type, check_name, domain_name, endpoint, status, start_time, error_text, additional_data) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := DB.Exec(query, event.MemberName, event.CheckType, event.CheckName, event.DomainName, event.Endpoint,
		event.Status, event.StartTime, event.ErrorText, event.AdditionalData)
	if err != nil {
		return 0, fmt.Errorf("failed to insert event: %w", err)
	}

	return result.LastInsertId()
}

// UpdateEvent sets the end time for an existing offline event.
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

// FindOpenOfflineEvent finds an existing open offline event for a given member, check type, and check name.
func FindOpenOfflineEvent(memberName, checkType, checkName, domainName, endpoint string) (*EventRecord, error) {
	var row *sql.Row

	if checkType == "endpoint" {
		query := `
		SELECT id, member_name, check_type, check_name, domain_name, endpoint, status, start_time, end_time, error_text, additional_data FROM member_events
		WHERE member_name = ? AND check_type = 'endpoint' AND check_name = ? AND domain_name = ? AND endpoint = ? AND status = FALSE AND end_time IS NULL
		`
		row = DB.QueryRow(query, memberName, checkName, domainName, endpoint)
	} else if checkType == "domain" {
		query := `
		SELECT id, member_name, check_type, check_name, domain_name, endpoint, status, start_time, end_time, error_text, additional_data FROM member_events
		WHERE member_name = ? AND check_type = 'domain' AND check_name = ? AND domain_name = ? AND status = FALSE AND end_time IS NULL
		`
		row = DB.QueryRow(query, memberName, checkName, domainName)
	} else if checkType == "site" {
		query := `
		SELECT id, member_name, check_type, check_name, domain_name, endpoint, status, start_time, end_time, error_text, additional_data FROM member_events
		WHERE member_name = ? AND check_type = 'site' AND check_name = ? AND status = FALSE AND end_time IS NULL
		`
		row = DB.QueryRow(query, memberName, checkName)
	}

	var event EventRecord
	err := row.Scan(&event.ID, &event.MemberName, &event.CheckType, &event.CheckName, &event.DomainName, &event.Endpoint,
		&event.Status, &event.StartTime, &event.EndTime, &event.ErrorText, &event.AdditionalData)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to find open offline event: %w", err)
	}
	return &event, nil
}

// GetEvents retrieves events for a member within the specified time range.
func GetEvents(memberName string, start, end time.Time) ([]EventRecord, error) {
	query := `SELECT id, member_name, check_type, check_name, domain_name, endpoint, status, start_time, end_time, error_text, additional_data
              FROM member_events
              WHERE member_name = ? AND start_time >= ? AND start_time <= ?`
  
	rows, err := DB.Query(query, memberName, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var res []EventRecord
	for rows.Next() {
		var ev EventRecord
		if err := rows.Scan(&ev.ID, &ev.MemberName, &ev.CheckType, &ev.CheckName, &ev.DomainName, &ev.Endpoint,
			&ev.Status, &ev.StartTime, &ev.EndTime, &ev.ErrorText, &ev.AdditionalData); err != nil {
			return nil, fmt.Errorf("scan error: %w", err)
		}
		res = append(res, ev)
	}
	return res, nil
}


// FetchEvents returns all events for the given member and optional domain within the specified time range.
func FetchEvents(memberName, domainName string, start, end time.Time) ([]EventRecord, error) {
	args := []interface{}{memberName, start, end}
	query := `
               SELECT id, member_name, check_type, check_name, domain_name, endpoint, status, start_time, end_time, error_text, additional_data FROM member_events
               WHERE member_name = ? AND start_time >= ? AND start_time <= ?`

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
		if err := rows.Scan(&e.ID, &e.MemberName, &e.CheckType, &e.CheckName, &e.DomainName, &e.Endpoint,
			&e.Status, &e.StartTime, &e.EndTime, &e.ErrorText, &e.AdditionalData); err != nil {
			return nil, fmt.Errorf("failed to scan event row: %w", err)
		}
		events = append(events, e)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return events, nil
}

