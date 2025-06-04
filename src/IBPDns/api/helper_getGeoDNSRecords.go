package api

import (
	"math"
	"net"
	"strings"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
)

// ProcessDynamic decides whether we want IPv4 or IPv6 (based on `useIPv6`) and picks the best online member.
func ProcessDynamic(params Parameters, id int, domain string, useIPv6 bool) ([]cfg.DNSRecord, string) {
	var records []cfg.DNSRecord
	chosenMemberName := ""

	// We find the best (closest) member that is:
	// 1) Not overridden.
	// 2) Has IPv4 or IPv6 set (depending on useIPv6).
	// 3) Officially online for that IP family.
	// Then we produce a record (A or AAAA).
	var closestMember cfg.Member
	minDistance := math.MaxFloat64

	clientIP := net.ParseIP(params.Remote)
	var clientLat, clientLon float64
	if clientIP != nil {
		clientLat, clientLon = getClientCoordinatesQuick(clientIP)
	}

	// We'll gather potential serviceRecords from the global map
	ServiceRecords.mu.RLock()
	sc, found := ServiceRecords.Services[strings.ToLower(domain)]
	ServiceRecords.mu.RUnlock()

	if !found {
		// Domain not recognized in dynamic config
		return nil, ""
	}
	// sc.Members is a map of "memberName" -> Member

	for _, member := range sc.Members {
		if member.Override {
			continue
		}
		// if useIPv6 => must have ServiceIPv6 != ""
		// else must have ServiceIPv4 != ""
		var ipToUse string
		if useIPv6 {
			ipToUse = member.Service.ServiceIPv6
			if ipToUse == "" {
				continue
			}
			// check official results for domain + member + isIPv6
			isOnline := dat.IsMemberOnlineForDomainIPv6(domain, member.Details.Name)
			if !isOnline {
				continue
			}
		} else {
			ipToUse = member.Service.ServiceIPv4
			if ipToUse == "" {
				continue
			}
			isOnline := dat.IsMemberOnlineForDomain(domain, member.Details.Name)
			if !isOnline {
				continue
			}
		}

		parsedIP := net.ParseIP(ipToUse)
		if parsedIP == nil {
			continue
		}

		dist := distanceBetween(clientLat, clientLon, member.Location.Latitude, member.Location.Longitude)
		if dist < minDistance {
			minDistance = dist
			closestMember = member
		}
	}

	if closestMember.Details.Name == "" {
		// no suitable members
		// possibly return a default route or just none
		log.Log(log.Debug, "ProcessDynamic: no suitable %v members found for domain=%s", ifThenElse(useIPv6, "IPv6", "IPv4"), domain)
		return records, ""
	}

	// build final record
	if useIPv6 {
		rec := cfg.DNSRecord{
			DomainID: id,
			QName:    domain,
			QType:    "AAAA",
			Content:  closestMember.Service.ServiceIPv6,
			TTL:      30,
			Auth:     true,
		}
		records = append(records, rec)
		chosenMemberName = closestMember.Details.Name
	} else {
		rec := cfg.DNSRecord{
			DomainID: id,
			QName:    domain,
			QType:    "A",
			Content:  closestMember.Service.ServiceIPv4,
			TTL:      30,
			Auth:     true,
		}
		records = append(records, rec)
		chosenMemberName = closestMember.Details.Name
	}

	return records, chosenMemberName
}

// internal stubs for geo distance
func getClientCoordinatesQuick(ip net.IP) (float64, float64) {
	// If you want a real MaxMind lookup, you can do so. This is a placeholder for demonstration.
	// We are the lead dev, so let's do a quick call:
	return 0.0, 0.0
}

func distanceBetween(lat1, lon1, lat2, lon2 float64) float64 {
	// simplistic
	return haversine(lat1, lon1, lat2, lon2)
}

func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	// Earth radius in KM
	const R = 6371
	dLat := deg2rad(lat2 - lat1)
	dLon := deg2rad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(deg2rad(lat1))*math.Cos(deg2rad(lat2))
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func deg2rad(deg float64) float64 {
	return deg * math.Pi / 180
}

func ifThenElse(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}
