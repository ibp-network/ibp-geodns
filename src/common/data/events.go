package data

import (
	"database/sql"
	"encoding/json"
	"time"

	"ibp-geodns/data/mysql"
	l "ibp-geodns/logging"
)

func RecordEvent(checkType, checkName, memberName, domainName, endpoint string, status bool, errorText string, data map[string]interface{}) {
	// Prepare data for storage
	var additionalData string
	if data != nil {
		dataBytes, _ := json.Marshal(data)
		additionalData = string(dataBytes)
	}

	// Handle events based on status
	if status {
		// Online Event: Close any existing offline event
		event, err := mysql.FindOpenOfflineEvent(memberName, checkType, checkName, domainName, endpoint)
		if err != nil {
			l.Log(l.Error, "Failed to check for existing offline event: %v", err)
			return
		}

		if event != nil {
			// Calculate the duration the member was offline
			now := time.Now().UTC()
			duration := now.Sub(event.StartTime)

			// If the duration is less than 30 seconds, delete the event
			if duration < 30*time.Second {
				err := mysql.DeleteEvent(event.ID)
				if err != nil {
					l.Log(l.Error, "Failed to delete short-duration event: %v", err)
				} else {
					l.Log(l.Info, "Deleted short-duration offline event for %s %s %s", memberName, checkType, checkName)
				}
				return
			}

			// Otherwise, update the event end time
			err = mysql.UpdateEventEndTime(event.ID, now)
			if err != nil {
				l.Log(l.Error, "Failed to update event end time: %v", err)
				return
			}
			l.Log(l.Info, "Closed offline event for %s %s %s", memberName, checkType, checkName)
		}
	} else {
		// Offline Event: Check if an open event already exists
		event, err := mysql.FindOpenOfflineEvent(memberName, checkType, checkName, domainName, endpoint)
		if err != nil {
			l.Log(l.Error, "Failed to check for existing offline event: %v", err)
			return
		}

		if event == nil {
			// Insert a new offline event
			_, err := mysql.InsertEvent(mysql.EventRecord{
				MemberName:     memberName,
				CheckType:      checkType,
				CheckName:      checkName,
				DomainName:     sql.NullString{String: domainName, Valid: domainName != ""},
				Endpoint:       sql.NullString{String: endpoint, Valid: endpoint != ""},
				Status:         false,
				StartTime:      time.Now().UTC(),
				ErrorText:      sql.NullString{String: errorText, Valid: errorText != ""},
				AdditionalData: sql.NullString{String: additionalData, Valid: additionalData != ""},
			})
			if err != nil {
				l.Log(l.Error, "Failed to insert offline event: %v", err)
			} else {
				l.Log(l.Info, "Recorded offline event for %s %s %s", memberName, checkType, checkName)
			}
		}
	}
}
