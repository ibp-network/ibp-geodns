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

	// (1) Set the global flags for saving caches:
	//     *This is the critical missing line so that SaveAllCaches() sees them.*
	SetCacheOptions(opts.UseLocalOfficialCaches, opts.UseUsageStats)

	// Initialize the global Stats struct
	Stats = &StatMap{Data: make(map[string]map[string]*DailyStats)}

	// If we have local/official caching, we load them now.
	// Then we force an immediate save so files appear quickly.
	if opts.UseLocalOfficialCaches {
		log.Log(log.Debug, "[data.Init] Loading local/official caches")
		LoadAllCaches()

		SaveAllCaches() // <-- new: force an immediate save so it re-creates the cache files now

		// auto-save official & local caches
		go startAutoUpdate()
	}

	// If usage is needed, we do usage-specific init.
	if opts.UseUsageStats {
		log.Log(log.Debug, "[data.Init] Enabling usage stats + daily usage processor")
		// load stats
		LoadAllCaches()

		// force an immediate save for stats too
		SaveAllCaches()

		// auto-save stats as well
		go startAutoUpdate()

		// start the daily usage aggregator
		go startDailyUsageProcessor()
	}
}

// MemberEnable sets the Override to false...
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

// MemberDisable sets the Override to true...
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

	// Check site-level results
	for _, sr := range sites {
		for _, r := range sr.Results {
			if r.Member.Details.Name == memberName && !r.Status {
				return false
			}
		}
	}

	// Check domain-level
	for _, dr := range domains {
		if dr.Domain == domain {
			for _, r := range dr.Results {
				if r.Member.Details.Name == memberName && !r.Status {
					return false
				}
			}
		}
	}

	// Check endpoint-level
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
