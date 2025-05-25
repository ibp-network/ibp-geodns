package data

import (
	cfg "ibp-geodns/src/common/config"
	"ibp-geodns/src/common/data/mysql"
	log "ibp-geodns/src/common/logging"
)

// InitOptions allows selective initialization of data subsystems.
type InitOptions struct {
	UseLocalOfficialCaches bool // if true, load/save local+official results
	UseUsageStats          bool // if true, track usage daily stats
}

// Init selectively initializes data subsystems based on InitOptions.
func Init(opts InitOptions) {
	log.Log(log.Debug, "[data.Init] Starting with options: %+v", opts)

	// Always initialize MySQL (for events, usage records, etc).
	go mysql.Init()

	// Initialize the global Stats struct
	Stats = &StatMap{Data: make(map[string]map[string]*DailyStats)}

	// If we have local/official caching, we load them now.
	// Then we force an immediate save so files appear quickly.
	if opts.UseLocalOfficialCaches {
		log.Log(log.Debug, "[data.Init] Loading local/official caches")
		LoadAllCaches()

		SaveAllCaches() // <-- new: force an immediate save so it re-creates the cache files now

		go startAutoUpdate() // auto-save official & local caches
	}

	// If usage is needed, we do usage-specific init.
	if opts.UseUsageStats {
		log.Log(log.Debug, "[data.Init] Enabling usage stats + daily usage processor")
		LoadAllCaches()

		SaveAllCaches() // <-- new: same idea for the stats cache

		go startAutoUpdate() // auto-save stats
		go startDailyUsageProcessor()
	}
}

// MemberEnable sets the Override to 1...
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

// MemberDisable sets the Override to 0...
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

// IsMemberOnlineForDomain checks official results...
func IsMemberOnlineForDomain(domain, memberName string) bool {
	sites, domains, endpoints := GetOfficialResults()

	for _, sr := range sites {
		for _, r := range sr.Results {
			if r.Member.Details.Name == memberName && !r.Status {
				return false
			}
		}
	}

	for _, dr := range domains {
		if dr.Domain == domain {
			for _, r := range dr.Results {
				if r.Member.Details.Name == memberName && !r.Status {
					return false
				}
			}
		}
	}

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
