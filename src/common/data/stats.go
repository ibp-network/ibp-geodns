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

// GetStats returns a copy of the IPv4 stats
func GetStats() map[string]map[string]*DailyStats {
	if Stats == nil {
		return nil
	}
	Stats.Mu.Lock()
	defer Stats.Mu.Unlock()

	return deepCopyStatMap(Stats.Data)
}

// GetStats6 returns a copy of the IPv6 stats
func GetStats6() map[string]map[string]*DailyStats {
	if Stats6 == nil {
		return nil
	}
	Stats6.Mu.Lock()
	defer Stats6.Mu.Unlock()

	return deepCopyStatMap(Stats6.Data)
}

func deepCopyStatMap(src map[string]map[string]*DailyStats) map[string]map[string]*DailyStats {
	copiedData := make(map[string]map[string]*DailyStats)
	for date, domains := range src {
		copiedData[date] = make(map[string]*DailyStats)
		for domain, domainStatPtr := range domains {
			if domainStatPtr == nil {
				continue
			}
			// Copy entire DailyStats
			clientStats := ClientStats{
				Requests:     domainStatPtr.ClientStats.Requests,
				ClassCs:      copyMap(domainStatPtr.ClientStats.ClassCs),
				Countries:    copyMap(domainStatPtr.ClientStats.Countries),
				Asns:         copyMap(domainStatPtr.ClientStats.Asns),
				Networks:     copyMap(domainStatPtr.ClientStats.Networks),
				CountryNames: copyMap(domainStatPtr.ClientStats.CountryNames),
			}
			memberStats := make(map[string]*MemberStats)
			for member, msPtr := range domainStatPtr.MemberStats {
				if msPtr == nil {
					continue
				}
				msCopy := &MemberStats{
					Requests:     msPtr.Requests,
					ClassCs:      copyMap(msPtr.ClassCs),
					Countries:    copyMap(msPtr.Countries),
					Asns:         copyMap(msPtr.Asns),
					Networks:     copyMap(msPtr.Networks),
					CountryNames: copyMap(msPtr.CountryNames),
				}
				memberStats[member] = msCopy
			}
			ds := &DailyStats{
				ClientStats: clientStats,
				MemberStats: memberStats,
			}
			copiedData[date][domain] = ds
		}
	}
	return copiedData
}

func copyMap(src map[string]int) map[string]int {
	dst := make(map[string]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// ClientHit records a client request for IPv4
func ClientHit(ReqIP string, ReqDomain string) {
	if Stats == nil {
		log.Log(log.Error, "Stats not initialized.")
		return
	}
	today := time.Now().UTC().Format("2006-01-02")
	countryCode := max.GetCountryCode(ReqIP)
	if countryCode == "" {
		countryCode = "Unknown"
	}
	classC := max.GetClassC(ReqIP)
	asn, networkName := max.GetAsnAndNetwork(ReqIP)
	countryName := max.GetCountryName(ReqIP)

	Stats.Mu.Lock()
	defer Stats.Mu.Unlock()

	if _, exists := Stats.Data[today]; !exists {
		Stats.Data[today] = make(map[string]*DailyStats)
	}
	if _, exists := Stats.Data[today][ReqDomain]; !exists {
		Stats.Data[today][ReqDomain] = &DailyStats{
			ClientStats: ClientStats{
				ClassCs:      make(map[string]int),
				Countries:    make(map[string]int),
				Asns:         make(map[string]int),
				Networks:     make(map[string]int),
				CountryNames: make(map[string]int),
			},
			MemberStats: make(map[string]*MemberStats),
		}
	}
	domainStats := Stats.Data[today][ReqDomain]

	domainStats.ClientStats.Requests++
	domainStats.ClientStats.ClassCs[classC]++
	domainStats.ClientStats.Countries[countryCode]++
	if asn != "" {
		domainStats.ClientStats.Asns[asn]++
	}
	if networkName != "" {
		domainStats.ClientStats.Networks[networkName]++
	}
	if countryName != "" {
		domainStats.ClientStats.CountryNames[countryName]++
	}
}

// ClientHitV6 records a client request for IPv6
func ClientHitV6(ReqIP string, ReqDomain string) {
	if Stats6 == nil {
		log.Log(log.Error, "Stats6 not initialized.")
		return
	}
	today := time.Now().UTC().Format("2006-01-02")
	countryCode := max.GetCountryCode(ReqIP)
	if countryCode == "" {
		countryCode = "Unknown"
	}
	// For IPv6, we won't store "classC"; we might store /64 or blank
	ipv6Trunc := "" // optional new function to get /64

	asn, networkName := max.GetAsnAndNetwork(ReqIP)
	countryName := max.GetCountryName(ReqIP)

	Stats6.Mu.Lock()
	defer Stats6.Mu.Unlock()

	if _, exists := Stats6.Data[today]; !exists {
		Stats6.Data[today] = make(map[string]*DailyStats)
	}
	if _, exists := Stats6.Data[today][ReqDomain]; !exists {
		Stats6.Data[today][ReqDomain] = &DailyStats{
			ClientStats: ClientStats{
				ClassCs:      make(map[string]int),
				Countries:    make(map[string]int),
				Asns:         make(map[string]int),
				Networks:     make(map[string]int),
				CountryNames: make(map[string]int),
			},
			MemberStats: make(map[string]*MemberStats),
		}
	}
	domainStats := Stats6.Data[today][ReqDomain]

	domainStats.ClientStats.Requests++
	if ipv6Trunc != "" {
		domainStats.ClientStats.ClassCs[ipv6Trunc]++
	}
	domainStats.ClientStats.Countries[countryCode]++
	if asn != "" {
		domainStats.ClientStats.Asns[asn]++
	}
	if networkName != "" {
		domainStats.ClientStats.Networks[networkName]++
	}
	if countryName != "" {
		domainStats.ClientStats.CountryNames[countryName]++
	}
}

// MemberHit increments usage for the assigned IPv4 member
func MemberHit(memberName string, ReqIP string, ReqDomain string) {
	if Stats == nil {
		log.Log(log.Error, "Stats not initialized.")
		return
	}
	today := time.Now().UTC().Format("2006-01-02")
	countryCode := max.GetCountryCode(ReqIP)
	if countryCode == "" {
		countryCode = "Unknown"
	}
	classC := max.GetClassC(ReqIP)
	asn, netName := max.GetAsnAndNetwork(ReqIP)
	countryName := max.GetCountryName(ReqIP)

	Stats.Mu.Lock()
	defer Stats.Mu.Unlock()

	if _, exists := Stats.Data[today]; !exists {
		Stats.Data[today] = make(map[string]*DailyStats)
	}
	if _, exists := Stats.Data[today][ReqDomain]; !exists {
		Stats.Data[today][ReqDomain] = &DailyStats{
			ClientStats: ClientStats{
				ClassCs:      make(map[string]int),
				Countries:    make(map[string]int),
				Asns:         make(map[string]int),
				Networks:     make(map[string]int),
				CountryNames: make(map[string]int),
			},
			MemberStats: make(map[string]*MemberStats),
		}
	}
	domainStats := Stats.Data[today][ReqDomain]

	if _, memberExists := domainStats.MemberStats[memberName]; !memberExists {
		domainStats.MemberStats[memberName] = &MemberStats{
			ClassCs:      make(map[string]int),
			Countries:    make(map[string]int),
			Asns:         make(map[string]int),
			Networks:     make(map[string]int),
			CountryNames: make(map[string]int),
		}
	}
	ms := domainStats.MemberStats[memberName]
	ms.Requests++
	ms.ClassCs[classC]++
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

// MemberHitV6 increments usage for the assigned IPv6 member
func MemberHitV6(memberName string, ReqIP string, ReqDomain string) {
	if Stats6 == nil {
		log.Log(log.Error, "Stats6 not initialized.")
		return
	}
	today := time.Now().UTC().Format("2006-01-02")
	countryCode := max.GetCountryCode(ReqIP)
	if countryCode == "" {
		countryCode = "Unknown"
	}
	ipv6Trunc := "" // optional prefix or blank
	asn, netName := max.GetAsnAndNetwork(ReqIP)
	countryName := max.GetCountryName(ReqIP)

	Stats6.Mu.Lock()
	defer Stats6.Mu.Unlock()

	if _, exists := Stats6.Data[today]; !exists {
		Stats6.Data[today] = make(map[string]*DailyStats)
	}
	if _, exists := Stats6.Data[today][ReqDomain]; !exists {
		Stats6.Data[today][ReqDomain] = &DailyStats{
			ClientStats: ClientStats{
				ClassCs:      make(map[string]int),
				Countries:    make(map[string]int),
				Asns:         make(map[string]int),
				Networks:     make(map[string]int),
				CountryNames: make(map[string]int),
			},
			MemberStats: make(map[string]*MemberStats),
		}
	}
	domainStats := Stats6.Data[today][ReqDomain]

	if _, memberExists := domainStats.MemberStats[memberName]; !memberExists {
		domainStats.MemberStats[memberName] = &MemberStats{
			ClassCs:      make(map[string]int),
			Countries:    make(map[string]int),
			Asns:         make(map[string]int),
			Networks:     make(map[string]int),
			CountryNames: make(map[string]int),
		}
	}
	ms := domainStats.MemberStats[memberName]
	ms.Requests++
	if ipv6Trunc != "" {
		ms.ClassCs[ipv6Trunc]++
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
