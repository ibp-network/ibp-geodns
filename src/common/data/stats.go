package data

import (
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
	"time"
)

var (
	Stats *StatMap
)

// GetStats returns a copy of the current statistics.
func GetStats() map[string]map[string]*DailyStats {
	if Stats == nil {
		return nil
	}

	Stats.Mu.Lock()
	defer Stats.Mu.Unlock()

	copiedData := make(map[string]map[string]*DailyStats)
	for date, domains := range Stats.Data {
		copiedData[date] = make(map[string]*DailyStats)
		for domain, domainStatPtr := range domains {
			if domainStatPtr == nil {
				continue
			}

			// Copy the entire DailyStats
			clientStats := ClientStats{
				Requests:  domainStatPtr.ClientStats.Requests,
				ClassCs:   copyMap(domainStatPtr.ClientStats.ClassCs),
				Countries: copyMap(domainStatPtr.ClientStats.Countries),

				// new
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
					Requests:  msPtr.Requests,
					ClassCs:   copyMap(msPtr.ClassCs),
					Countries: copyMap(msPtr.Countries),
					// new
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

// copyMap is a small helper to clone map[string]int
func copyMap(src map[string]int) map[string]int {
	dst := make(map[string]int, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// ClientHit records a client request for a given IP and Domain.
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

	// NEW: get asn, network, country name
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

	// increment new fields
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

// MemberHit records that a member served the request to the client.
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

	// NEW: get asn, network, country name
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

	// If nil, create a new MemberStats
	if _, memberExists := domainStats.MemberStats[memberName]; !memberExists {
		domainStats.MemberStats[memberName] = &MemberStats{
			ClassCs:      make(map[string]int),
			Countries:    make(map[string]int),
			Asns:         make(map[string]int),
			Networks:     make(map[string]int),
			CountryNames: make(map[string]int),
		}
	}

	memberStats := domainStats.MemberStats[memberName]

	memberStats.Requests++
	memberStats.ClassCs[classC]++
	memberStats.Countries[countryCode]++

	// increment new fields
	if asn != "" {
		memberStats.Asns[asn]++
	}
	if networkName != "" {
		memberStats.Networks[networkName]++
	}
	if countryName != "" {
		memberStats.CountryNames[countryName]++
	}
}
