package data

import (
	"database/sql"
	"encoding/json"
	"time"

	"ibp-geodns/src/common/data/mysql"
	log "ibp-geodns/src/common/logging"
)

func RecordEvent(checkType, checkName, memberName, domainName, endpoint string, status bool, errorText string, data map[string]interface{}, isIPv6 bool) {
	var additionalData string
	if data != nil {
		dataBytes, _ := json.Marshal(data)
		additionalData = string(dataBytes)
	}

	if status {
		event, err := mysql.FindOpenOfflineEvent(memberName, checkType, checkName, domainName, endpoint, isIPv6)
		if err != nil {
			log.Log(log.Error, "Failed to check for existing offline event: %v", err)
			return
		}
		if event != nil {
			now := time.Now().UTC()
			duration := now.Sub(event.StartTime)
			if duration < 30*time.Second {
				err := mysql.DeleteEvent(event.ID)
				if err != nil {
					log.Log(log.Error, "Failed to delete short-duration event: %v", err)
				} else {
					log.Log(log.Info, "Deleted short-duration offline event for %s %s %s isIPv6=%v", memberName, checkType, checkName, isIPv6)
				}
				return
			}
			err = mysql.UpdateEventEndTime(event.ID, now)
			if err != nil {
				log.Log(log.Error, "Failed to update event end time: %v", err)
				return
			}
			log.Log(log.Info, "Closed offline event for %s %s %s isIPv6=%v", memberName, checkType, checkName, isIPv6)
		}
	} else {
		event, err := mysql.FindOpenOfflineEvent(memberName, checkType, checkName, domainName, endpoint, isIPv6)
		if err != nil {
			log.Log(log.Error, "Failed to check for existing offline event: %v", err)
			return
		}
		if event == nil {
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
				IsIPv6:         isIPv6,
			})
			if err != nil {
				log.Log(log.Error, "Failed to insert offline event: %v", err)
			} else {
				log.Log(log.Info, "Recorded offline event for %s %s %s isIPv6=%v", memberName, checkType, checkName, isIPv6)
			}
		}
	}
}

func GetMemberEvents(memberName, domain string, start, end time.Time) ([]EventRecord, error) {
	rows, err := mysql.FetchEvents(memberName, domain, start, end)
	if err != nil {
		return nil, err
	}

	events := make([]EventRecord, 0, len(rows))
	for _, r := range rows {
		var dataMap map[string]interface{}
		if r.AdditionalData.Valid && r.AdditionalData.String != "" {
			_ = json.Unmarshal([]byte(r.AdditionalData.String), &dataMap)
		}
		var domainName, endpoint, errText string
		if r.DomainName.Valid {
			domainName = r.DomainName.String
		}
		if r.Endpoint.Valid {
			endpoint = r.Endpoint.String
		}
		if r.ErrorText.Valid {
			errText = r.ErrorText.String
		}

		var endTime time.Time
		if r.EndTime.Valid {
			endTime = r.EndTime.Time
		}

		events = append(events, EventRecord{
			CheckType:  r.CheckType,
			CheckName:  r.CheckName,
			MemberName: r.MemberName,
			DomainName: domainName,
			Endpoint:   endpoint,
			Status:     r.Status,
			ErrorText:  errText,
			Data:       dataMap,
			StartTime:  r.StartTime,
			EndTime:    endTime,
			StartDate:  r.StartTime.Format("2006-01-02"),
			EndDate:    endTime.Format("2006-01-02"),
			IsIPv6:     r.IsIPv6,
		})
	}
	return events, nil
}
