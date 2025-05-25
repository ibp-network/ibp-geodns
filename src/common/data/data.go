package data

import (
	cfg "ibp-geodns/src/common/config"
	"ibp-geodns/src/common/data/mysql"
	log "ibp-geodns/src/common/logging"
)

// InitOptions allows selective initialization of data subsystems.
// For example, a serviceMonitor node might only need local/official caching
// but not usage stats. A dnsApi node might want usage stats but not local/official caching.
type InitOptions struct {
	UseLocalOfficialCaches bool // if true, load/save local+official results
	UseUsageStats          bool // if true, track usage daily stats
}

// Init selectively initializes data subsystems based on InitOptions.
func Init(opts InitOptions) {
	log.Log(log.Debug, "[data.Init] Starting with options: %+v", opts)

	// Always initialize MySQL (for events, usage records, etc).
	go mysql.Init()

	// Initialize the global Stats struct (it won’t hurt to always have it).
	// If UseUsageStats is false, we simply won’t do daily usage processing.
	Stats = &StatMap{Data: make(map[string]map[string]*DailyStats)}

	// If we have local/official caching, we load them now & start auto-saves.
	if opts.UseLocalOfficialCaches {
		log.Log(log.Debug, "[data.Init] Loading local/official caches")
		LoadAllCaches()
		go startAutoUpdate() // auto-save official & local caches
	}

	// If usage is needed, we do usage-specific init (like daily usage).
	if opts.UseUsageStats {
		log.Log(log.Debug, "[data.Init] Enabling usage stats + daily usage processor")
		// If your usage stats rely on the same cache system, you can reuse LoadAllCaches()
		// or separate them out. For example, if you want the stats.cache.json to load:
		LoadAllCaches()
		go startAutoUpdate() // if you want the stats also auto-saved
		go startDailyUsageProcessor()
	}
}

// MemberEnable sets the Override to 1 for the specified member name and triggers an event.
func MemberEnable(name string) {
	member, exists := cfg.GetMember(name)
	if !exists {
		log.Log(log.Debug, "Could not enable member; does not exist")
		return
	}

	member.Override = false
	cfg.SetMember(name, member)
	RecordEvent("site", "MemberEnable", name, "", "", true, "Member has disabled override.", nil)
}

// MemberDisable sets the Override to 0 for the specified member name and triggers an event.
func MemberDisable(name string) {
	member, exists := cfg.GetMember(name)
	if !exists {
		log.Log(log.Debug, "Could not disable member; does not exist")
		return
	}

	member.Override = true
	cfg.SetMember(name, member)
	RecordEvent("site", "MemberDisable", name, "", "", false, "Member has enabled override.", nil)
}

// IsMemberOnlineForDomain checks official results to see if a member is online for a given domain.
func IsMemberOnlineForDomain(domain, memberName string) bool {
	sites, domains, endpoints := GetOfficialResults()

	// 1) Check all site results for this member.
	for _, sr := range sites {
		for _, r := range sr.Results {
			if r.Member.Details.Name == memberName && !r.Status {
				return false
			}
		}
	}

	// 2) If site checks are good, check domain-level results.
	for _, dr := range domains {
		if dr.Domain == domain {
			for _, r := range dr.Results {
				if r.Member.Details.Name == memberName && !r.Status {
					return false
				}
			}
		}
	}

	// 3) Check endpoint results related to that domain.
	for _, er := range endpoints {
		if er.Domain == domain {
			for _, r := range er.Results {
				if r.Member.Details.Name == memberName && !r.Status {
					return false
				}
			}
		}
	}

	return true
}
