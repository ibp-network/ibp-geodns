package api

import (
	"net/http"
	"time"
)

func handle_GetAllDomains(w http.ResponseWriter, r *http.Request, req Request) Response {
	currentUnixTimestamp := int(time.Now().UTC().Unix())
	records := []DomainInfo{}
	for key, domain := range TLDRecords.records {
		Masters := []string{}

		// Define the prefixes for the DNS servers
		dnsPrefixes := []string{"dns-01", "dns-02", "dns-03"}

		// Iterate over the prefixes to construct the full domain names
		for _, prefix := range dnsPrefixes {
			dnsName := prefix + "." + domain

			// Search for the DNSRecord in staticEntries
			for _, dnsRecord := range StaticRecords.records {
				if dnsRecord.QName == dnsName {
					Masters = append(Masters, dnsRecord.Content)
					break
				}
			}
		}

		records = append(records, DomainInfo{
			DomainID:       key,
			Zone:           domain,
			Masters:        Masters,
			NotifiedSerial: currentUnixTimestamp,
			Serial:         currentUnixTimestamp,
			LastCheck:      currentUnixTimestamp,
			Kind:           "NATIVE",
		})
	}

	return Response{Result: records}
}
