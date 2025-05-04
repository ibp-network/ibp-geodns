package data

import (
	"common/config"
	"common/data/mysql"
	l "common/logging"
)

func Init() {
	Stats = &StatMap{Data: make(map[string]map[string]*DailyStats)}
	LoadAllCaches()
	go startAutoUpdate()
	go mysql.Init()
}

// MemberEnable sets the Override to 1 for the specified member name, stores it in MySQL, and triggers an event.
func MemberEnable(name string) {
	member, exists := config.GetMember(name)
	if !exists {
		l.Log(l.Debug, "Could not enable member does not exist")
		return
	}

	member.Override = false
	config.SetMember(name, member)
	RecordEvent("site", "MemberEnable", name, "", "", true, "Member has disabled override.", nil)
}

// MemberDisable sets the Override to 0 for the specified member name, stores it in MySQL, and triggers an event.
func MemberDisable(name string) {
	member, exists := config.GetMember(name)
	if !exists {
		l.Log(l.Debug, "Could not enable member does not exist")
		return
	}

	member.Override = true
	config.SetMember(name, member)
	RecordEvent("site", "MemberEnable", name, "", "", false, "Member has enabled override.", nil)
}

func IsMemberOnlineForDomain(domain, memberName string) bool {
	sites, domains, endpoints := GetOfficialResults()

	// Check all site results for memberName. If any fail => offline.
	for _, sr := range sites {
		for _, r := range sr.Results {
			if r.Member.Details.Name == memberName && !r.Status {
				return false
			}
		}
	}

	// If site checks are good, check domain-level checks for memberName and domain.
	for _, dr := range domains {
		if dr.Domain == domain {
			for _, r := range dr.Results {
				if r.Member.Details.Name == memberName && !r.Status {
					return false
				}
			}
		}
	}

	// Check endpoint results related to that domain
	for _, er := range endpoints {
		if er.Domain == domain {
			for _, r := range er.Results {
				if r.Member.Details.Name == memberName && !r.Status {
					return false
				}
			}
		}
	}

	// If we reach this point, no failures were found, so the member is considered online.
	return true
}
