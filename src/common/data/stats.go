package data

import (
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
	"time"
)

var (
	Stats  *StatMap
	Stats6 *StatMap // for IPv6 usage
)

// We remove ClientHit, MemberHit, and their IPv6 variants, merging them into RecordDnsHit.

// RecordDnsHit merges "client" and "member" counters into a single usage increment.
// If memberName is empty, we store usage under "(none)" so we still track domain requests
// that didn't assign a member. If memberName is non-empty, we store usage under that member.
func RecordDnsHit(isIPv6 bool, clientIP, domain, memberName string) {
	if domain == "" || clientIP == "" {
		return
	}

	now := time.Now().UTC()
	date := now.Format("2006-01-02") // Example: "2025-05-28"

	countryCode := max.GetCountryCode(clientIP)
	if countryCode == "" {
		countryCode = "Unknown"
	}
	var classC string
	if isIPv6 {
		classC = "" // Optionally store /64 if you want to handle IPv6 subnets
	} else {
		classC = max.GetClassC(clientIP)
	}
	asn, netName := max.GetAsnAndNetwork(clientIP)
	countryName := max.GetCountryName(clientIP)

	if memberName == "" {
		memberName = "(none)"
	}

	// Choose Stats vs Stats6
	if isIPv6 {
		Stats6.Mu.Lock()
		defer Stats6.Mu.Unlock()

		if _, exists := Stats6.Data[date]; !exists {
			Stats6.Data[date] = make(map[string]*DailyStats)
		}
		if _, exists := Stats6.Data[date][domain]; !exists {
			Stats6.Data[date][domain] = &DailyStats{
				MemberStats: make(map[string]*MemberStats),
			}
		}

		dstats := Stats6.Data[date][domain]
		if _, exists := dstats.MemberStats[memberName]; !exists {
			dstats.MemberStats[memberName] = &MemberStats{
				ClassCs:      make(map[string]int),
				Countries:    make(map[string]int),
				Asns:         make(map[string]int),
				Networks:     make(map[string]int),
				CountryNames: make(map[string]int),
			}
		}

		ms := dstats.MemberStats[memberName]
		ms.Requests++
		if classC != "" {
			ms.ClassCs[classC]++
		}
		ms.Countries[countryCode]++
		if asn != "" {
			ms.Asns[asn]++
		}
		if netName != "" {
			ms.Networks[netName]++
		}
		if countryName != "" {
			ms.CountryNames[countryName]++
		}

	} else {
		Stats.Mu.Lock()
		defer Stats.Mu.Unlock()

		if _, exists := Stats.Data[date]; !exists {
			Stats.Data[date] = make(map[string]*DailyStats)
		}
		if _, exists := Stats.Data[date][domain]; !exists {
			Stats.Data[date][domain] = &DailyStats{
				MemberStats: make(map[string]*MemberStats),
			}
		}

		dstats := Stats.Data[date][domain]
		if _, exists := dstats.MemberStats[memberName]; !exists {
			dstats.MemberStats[memberName] = &MemberStats{
				ClassCs:      make(map[string]int),
				Countries:    make(map[string]int),
				Asns:         make(map[string]int),
				Networks:     make(map[string]int),
				CountryNames: make(map[string]int),
			}
		}

		ms := dstats.MemberStats[memberName]
		ms.Requests++
		if classC != "" {
			ms.ClassCs[classC]++
		}
		ms.Countries[countryCode]++
		if asn != "" {
			ms.Asns[asn]++
		}
		if netName != "" {
			ms.Networks[netName]++
		}
		if countryName != "" {
			ms.CountryNames[countryName]++
		}
	}
	log.Log(log.Debug, "[RecordDnsHit] domain=%s, member=%s, isIPv6=%v", domain, memberName, isIPv6)
}
