package api

import (
	"math"
	"net"
	"strings"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
)

func ProcessDynamic(params Parameters, id int, domain string, useIPv6 bool) ([]cfg.DNSRecord, string) {
	var records []cfg.DNSRecord
	chosenMemberName := ""

	var closestMember cfg.Member
	minDistance := math.MaxFloat64

	clientIP := net.ParseIP(params.Remote)
	var clientLat, clientLon float64
	if clientIP != nil {
		isClientIPv6 := clientIP.To4() == nil

		if isClientIPv6 {
			log.Log(log.Debug, "ProcessDynamic: Client is using IPv6: %s", clientIP.String())
			clientLat, clientLon = max.GetClientCoordinates(clientIP.String())
		} else {
			log.Log(log.Debug, "ProcessDynamic: Client is using IPv4: %s", clientIP.String())
			clientLat, clientLon = max.GetClientCoordinates(clientIP.String())
		}
	}

	ServiceRecords.mu.RLock()
	sc, found := ServiceRecords.Services[strings.ToLower(domain)]
	ServiceRecords.mu.RUnlock()

	if !found {
		return nil, ""
	}

	for _, member := range sc.Members {
		if member.Override {
			continue
		}

		var ipToUse string
		if useIPv6 {
			ipToUse = member.Service.ServiceIPv6
			if ipToUse == "" {
				continue
			}

			isOnline := IsMemberOnlineForDomainIPv6(domain, member.Details.Name)
			if !isOnline {
				log.Log(log.Debug, "ProcessDynamic: member %s is offline for IPv6 on domain %s",
					member.Details.Name, domain)
				continue
			}
		} else {
			ipToUse = member.Service.ServiceIPv4
			if ipToUse == "" {
				continue
			}

			isOnline := IsMemberOnlineForDomainIPv4(domain, member.Details.Name)
			if !isOnline {
				log.Log(log.Debug, "ProcessDynamic: member %s is offline for IPv4 on domain %s",
					member.Details.Name, domain)
				continue
			}
		}

		parsedIP := net.ParseIP(ipToUse)
		if parsedIP == nil {
			continue
		}

		dist := max.Distance(clientLat, clientLon, member.Location.Latitude, member.Location.Longitude)
		if dist < minDistance {
			minDistance = dist
			closestMember = member
		}
	}

	if closestMember.Details.Name == "" {
		log.Log(log.Debug, "ProcessDynamic: no suitable %v members found for domain=%s", boolToStr(useIPv6), domain)
		return records, ""
	}

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

		dat.RecordDnsHit(true, params.Remote, domain, chosenMemberName)
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

		dat.RecordDnsHit(false, params.Remote, domain, chosenMemberName)
	}

	log.Log(log.Debug, "ProcessDynamic: selected member %s for %s query on domain %s",
		chosenMemberName, boolToStr(useIPv6), domain)

	return records, chosenMemberName
}

func boolToStr(b bool) string {
	if b {
		return "IPv6"
	}
	return "IPv4"
}
