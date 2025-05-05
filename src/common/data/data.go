package data

import (
	cfg "ibp-geodns/src/common/config"
	"ibp-geodns/src/common/data/mysql"
	log "ibp-geodns/src/common/logging"
)

func Init() {
	Stats = &StatMap{Data: make(map[string]map[string]*DailyStats)}
	LoadAllCaches()
	go startAutoUpdate()
	go mysql.Init()
}

// MemberEnable sets the Override to 1 for the specified member name, stores it in MySQL, and triggers an event.
func MemberEnable(name string) {
	member, exists := cfg.GetMember(name)
	if !exists {
		log.Log(log.Debug, "Could not enable member does not exist")
		return
	}

	member.Override = false
	cfg.SetMember(name, member)
	RecordEvent("site", "MemberEnable", name, "", "", true, "Member has disabled override.", nil)
}

// MemberDisable sets the Override to 0 for the specified member name, stores it in MySQL, and triggers an event.
func MemberDisable(name string) {
	member, exists := cfg.GetMember(name)
	if !exists {
		log.Log(log.Debug, "Could not enable member does not exist")
		return
	}

	member.Override = true
	cfg.SetMember(name, member)
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
