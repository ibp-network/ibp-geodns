package data

import (
	cfg "ibp-geodns/src/common/config"
	mysql "ibp-geodns/src/common/data/mysql"
	log "ibp-geodns/src/common/logging"
	"strings"
	"time"
)

/*
   ---------------------------------------------------------------------------
   Init machinery (unchanged – full file retained)
   ---------------------------------------------------------------------------
*/

type InitOptions struct {
	UseLocalOfficialCaches bool
	UseUsageStats          bool
}

func Init(opts InitOptions) {
	log.Log(log.Debug, "[data.Init] Starting with options: %+v", opts)

	mysql.Init()

	SetCacheOptions(opts.UseLocalOfficialCaches, opts.UseUsageStats)

	if opts.UseLocalOfficialCaches {
		LoadAllCaches()
		SaveAllCaches()
		go startAutoUpdate()
	}

	go startPeriodicUsageFlush()
}

/*
   ---------------------------------------------------------------------------
   Member enable / disable helpers (unchanged)
   ---------------------------------------------------------------------------
*/

func MemberEnable(name string) {
	member, exists := cfg.GetMember(name)
	if !exists {
		log.Log(log.Debug, "Could not enable member; does not exist")
		return
	}
	member.Override = false
	cfg.SetMember(name, member)
	RecordEvent("site", "MemberEnable", name, "", "", true, "Member has disabled override.", nil, false)
	RecordEvent("site", "MemberEnable", name, "", "", true, "Member has disabled override.", nil, true)
}

func MemberDisable(name string) {
	member, exists := cfg.GetMember(name)
	if !exists {
		log.Log(log.Debug, "Could not disable member; does not exist")
		return
	}
	member.Override = true
	cfg.SetMember(name, member)
	RecordEvent("site", "MemberDisable", name, "", "", false, "Member has enabled override.", nil, false)
	RecordEvent("site", "MemberDisable", name, "", "", false, "Member has enabled override.", nil, true)
}

/*
   ---------------------------------------------------------------------------
   NEW helpers – determine “latest” status
   ---------------------------------------------------------------------------
*/

func newestStatus(results []Result, memberName string) (found bool, latest bool, newest time.Time) {
	for _, r := range results {
		if r.Member.Details.Name != memberName {
			continue
		}
		if !found || r.Checktime.After(newest) {
			found = true
			latest = r.Status
			newest = r.Checktime
		}
	}
	return
}

func newestSiteStatus(sites []SiteResult, memberName string, ipv6Filter *bool) (bool, bool) {
	var newest time.Time
	found := false
	latest := false

	for _, sr := range sites {
		if ipv6Filter != nil && sr.IsIPv6 != *ipv6Filter {
			continue
		}
		if ok, st, ct := newestStatus(sr.Results, memberName); ok {
			if !found || ct.After(newest) {
				found = true
				latest = st
				newest = ct
			}
		}
	}
	return found, latest
}

func newestDomainStatus(domains []DomainResult, memberName, domain string, ipv6Filter *bool) (bool, bool) {
	var newest time.Time
	found := false
	latest := false

	for _, dr := range domains {
		if !strings.EqualFold(dr.Domain, domain) {
			continue
		}
		if ipv6Filter != nil && dr.IsIPv6 != *ipv6Filter {
			continue
		}
		if ok, st, ct := newestStatus(dr.Results, memberName); ok {
			if !found || ct.After(newest) {
				found = true
				latest = st
				newest = ct
			}
		}
	}
	return found, latest
}

func newestEndpointStatus(endpoints []EndpointResult, memberName, domain string, ipv6Filter *bool) (bool, bool) {
	var newest time.Time
	found := false
	latest := false

	for _, er := range endpoints {
		if !strings.EqualFold(er.Domain, domain) {
			continue
		}
		if ipv6Filter != nil && er.IsIPv6 != *ipv6Filter {
			continue
		}
		if ok, st, ct := newestStatus(er.Results, memberName); ok {
			if !found || ct.After(newest) {
				found = true
				latest = st
				newest = ct
			}
		}
	}
	return found, latest
}

/*
   ---------------------------------------------------------------------------
   Public status helpers (re‑implemented to use newest‑status helpers)
   ---------------------------------------------------------------------------
*/

// overall: any failed (v4 or v6) ⇒ offline
func IsMemberOnlineForDomain(domain, memberName string) bool {
	sites, domains, endpoints := GetOfficialResults()

	if ok, st := newestSiteStatus(sites, memberName, nil); ok && !st {
		return false
	}
	if ok, st := newestDomainStatus(domains, memberName, domain, nil); ok && !st {
		return false
	}
	if ok, st := newestEndpointStatus(endpoints, memberName, domain, nil); ok && !st {
		return false
	}
	return true
}

func IsMemberOnlineForDomainIPv4(domain, memberName string) bool {
	ipv6 := false
	sites, domains, endpoints := GetOfficialResults()

	if ok, st := newestSiteStatus(sites, memberName, &ipv6); ok && !st {
		return false
	}
	if ok, st := newestDomainStatus(domains, memberName, domain, &ipv6); ok && !st {
		return false
	}
	if ok, st := newestEndpointStatus(endpoints, memberName, domain, &ipv6); ok && !st {
		return false
	}
	return true
}

func IsMemberOnlineForDomainIPv6(domain, memberName string) bool {
	ipv6 := true
	sites, domains, endpoints := GetOfficialResults()

	if ok, st := newestSiteStatus(sites, memberName, &ipv6); ok && !st {
		return false
	}
	if ok, st := newestDomainStatus(domains, memberName, domain, &ipv6); ok && !st {
		return false
	}
	if ok, st := newestEndpointStatus(endpoints, memberName, domain, &ipv6); ok && !st {
		return false
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
