package data

import (
	l "ibp-geodns/logging"
	g "ibp-geodns/networking/geoip"
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

			clientStats := ClientStats{
				Requests:  domainStatPtr.ClientStats.Requests,
				ClassCs:   make(map[string]int),
				Countries: make(map[string]int),
			}
			for k, v := range domainStatPtr.ClientStats.ClassCs {
				clientStats.ClassCs[k] = v
			}
			for k, v := range domainStatPtr.ClientStats.Countries {
				clientStats.Countries[k] = v
			}

			memberStats := make(map[string]*MemberStats)
			for member, msPtr := range domainStatPtr.MemberStats {
				if msPtr == nil {
					continue
				}
				msCopy := &MemberStats{
					Requests:  msPtr.Requests,
					ClassCs:   make(map[string]int),
					Countries: make(map[string]int),
				}
				for ck, cv := range msPtr.ClassCs {
					msCopy.ClassCs[ck] = cv
				}
				for ck, cv := range msPtr.Countries {
					msCopy.Countries[ck] = cv
				}
				memberStats[member] = msCopy
			}

			copiedData[date][domain] = &DailyStats{
				ClientStats: clientStats,
				MemberStats: memberStats,
			}
		}
	}

	return copiedData
}

// ClientHit records a client request for a given IP and Domain.
func ClientHit(ReqIP string, ReqDomain string) {
	if Stats == nil {
		l.Log(l.Error, "Stats not initialized. Call InitStats first.")
		return
	}

	today := time.Now().UTC().Format("2006-01-02")
	countryCode := g.GetCountryCode(ReqIP)
	if countryCode == "" {
		countryCode = "Unknown"
	}
	classC := g.GetClassC(ReqIP)

	Stats.Mu.Lock()
	defer Stats.Mu.Unlock()

	if _, exists := Stats.Data[today]; !exists {
		Stats.Data[today] = make(map[string]*DailyStats)
	}
	if _, exists := Stats.Data[today][ReqDomain]; !exists {
		Stats.Data[today][ReqDomain] = &DailyStats{
			ClientStats: ClientStats{
				ClassCs:   make(map[string]int),
				Countries: make(map[string]int),
				Requests:  0,
			},
			MemberStats: make(map[string]*MemberStats),
		}
	}

	domainStats := Stats.Data[today][ReqDomain]
	domainStats.ClientStats.Requests++
	domainStats.ClientStats.ClassCs[classC]++
	domainStats.ClientStats.Countries[countryCode]++
}

// MemberHit records that a member served the request to the client.
func MemberHit(memberName string, ReqIP string, ReqDomain string) {
	if Stats == nil {
		l.Log(l.Error, "Stats not initialized. Call InitStats first.")
		return
	}

	today := time.Now().UTC().Format("2006-01-02")
	countryCode := g.GetCountryCode(ReqIP)
	if countryCode == "" {
		countryCode = "Unknown"
	}
	classC := g.GetClassC(ReqIP)

	Stats.Mu.Lock()
	defer Stats.Mu.Unlock()

	if _, exists := Stats.Data[today]; !exists {
		Stats.Data[today] = make(map[string]*DailyStats)
	}

	if _, exists := Stats.Data[today][ReqDomain]; !exists {
		Stats.Data[today][ReqDomain] = &DailyStats{
			ClientStats: ClientStats{
				ClassCs:   make(map[string]int),
				Countries: make(map[string]int),
				Requests:  0,
			},
			MemberStats: make(map[string]*MemberStats),
		}
	}

	domainStats := Stats.Data[today][ReqDomain]

	if domainStats.MemberStats == nil {
		domainStats.MemberStats = make(map[string]*MemberStats)
	}
	if _, exists := domainStats.MemberStats[memberName]; !exists {
		domainStats.MemberStats[memberName] = &MemberStats{
			ClassCs:   make(map[string]int),
			Countries: make(map[string]int),
			Requests:  0,
		}
	}

	memberStats := domainStats.MemberStats[memberName]
	memberStats.Requests++
	memberStats.ClassCs[classC]++
	memberStats.Countries[countryCode]++
}
