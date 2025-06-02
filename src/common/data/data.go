package data

import (
	cfg "ibp-geodns/src/common/config"
	mysql "ibp-geodns/src/common/data/mysql"
	log "ibp-geodns/src/common/logging"
	"time"
)

// InitOptions allows selective initialization of data subsystems (caches, etc.).
// The usage stats flush is now always started unconditionally.
type InitOptions struct {
	UseLocalOfficialCaches bool // if true, load/save local+official results
	UseUsageStats          bool // if true, track usage daily stats (for future checks)
}

// Init selectively initializes data subsystems based on InitOptions.
func Init(opts InitOptions) {
	log.Log(log.Debug, "[data.Init] Starting with options: %+v", opts)

	// Always initialize MySQL (for events, usage records, etc.).
	go mysql.Init()

	// (1) Set the global flags for saving caches:
	SetCacheOptions(opts.UseLocalOfficialCaches, opts.UseUsageStats)

	// Load caches if needed (local official caches).
	if opts.UseLocalOfficialCaches {
		LoadAllCaches()
		SaveAllCaches()
		// auto-save official & local caches periodically
		go startAutoUpdate()
	}

	// (2) Always start the periodic usage flush every 5 minutes, regardless of UseUsageStats.
	// The daily flush for "yesterday" is removed since we flush frequently.
	go startPeriodicUsageFlush()
}

// MemberEnable sets Override=false on a member and records an event.
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

// MemberDisable sets Override=true on a member and records an event.
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

// IsMemberOnlineForDomain checks official results for IPv4.
func IsMemberOnlineForDomain(domain, memberName string) bool {
	sites, domains, endpoints := GetOfficialResults()

	// site-level
	for _, sr := range sites {
		for _, r := range sr.Results {
			if r.Member.Details.Name == memberName && !r.Status {
				return false
			}
		}
	}

	// domain-level
	for _, dr := range domains {
		if dr.Domain == domain {
			for _, r := range dr.Results {
				if r.Member.Details.Name == memberName && !r.Status {
					return false
				}
			}
		}
	}

	// endpoint-level
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

// IsMemberOnlineForDomainIPv6 checks official results for IPv6.
func IsMemberOnlineForDomainIPv6(domain, memberName string) bool {
	sites, domains, endpoints := GetOfficialResults()

	// site-level (only if IsIPv6 == true)
	for _, sr := range sites {
		if !sr.IsIPv6 {
			continue
		}
		for _, r := range sr.Results {
			if r.Member.Details.Name == memberName && !r.Status {
				return false
			}
		}
	}

	// domain-level (only if IsIPv6 == true)
	for _, dr := range domains {
		if !dr.IsIPv6 {
			continue
		}
		if dr.Domain == domain {
			for _, r := range dr.Results {
				if r.Member.Details.Name == memberName && !r.Status {
					return false
				}
			}
		}
	}

	// endpoint-level (only if IsIPv6 == true)
	for _, er := range endpoints {
		if !er.IsIPv6 {
			continue
		}
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

// startAutoUpdate periodically calls SaveAllCaches() so we keep disk caches updated.
func startAutoUpdate() {
	ticker := time.NewTicker(90 * time.Second)
	go func() {
		for range ticker.C {
			SaveAllCaches()
		}
	}()
}

// startPeriodicUsageFlush flushes the current day's usage to the DB every 5 minutes.
func startPeriodicUsageFlush() {
	ticker := time.NewTicker(5 * time.Minute)
	for {
		<-ticker.C
		today := time.Now().UTC().Format("2006-01-02")
		log.Log(log.Info, "[startPeriodicUsageFlush] Flushing usage for today: %s", today)
		FlushUsageToDatabase(today)
	}
}
