package api

import (
	"time"

	log "github.com/ibp-network/ibp-geodns-libs/logging"
)

func handle_GetDomainInfo(req Request) Response {
	name := req.Parameters.Name
	if name == "" {
		name = req.Parameters.QName
	}

	log.Log(log.Debug, "handle_GetDomainInfo: name=%s", name)

	id, domain, ok := findZone(name)
	if !ok {
		return Response{Result: nil}
	}

	currentUnixTimestamp := int(time.Now().UTC().Unix())
	record := DomainInfo{
		DomainID:       id,
		Zone:           domain,
		Masters:        gatherMasters(domain),
		NotifiedSerial: currentUnixTimestamp,
		Serial:         currentUnixTimestamp,
		LastCheck:      currentUnixTimestamp,
		Kind:           "NATIVE",
	}

	log.Log(log.Debug, "handle_GetDomainInfo: returning zone=%s", domain)
	return Response{Result: record}
}
