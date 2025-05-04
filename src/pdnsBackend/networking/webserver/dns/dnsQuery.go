package dns

import (
	"ibp-geodns/src/common/config"
	"net/http"
	"time"
)

// Handle initialization call (not necessary for us to do anything)
func DnsQuery_Initialize(w http.ResponseWriter, r *http.Request, req Request) Response {
	return Response{Result: true}
}

func DnsQuery_GetAllDomains(w http.ResponseWriter, r *http.Request, req Request) Response {
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

func DnsQuery_GetDomainInfo(w http.ResponseWriter, r *http.Request, req Request) Response {
	var records []DomainInfo
	currentUnixTimestamp := int(time.Now().UTC().Unix())

	for key, domain := range TLDRecords.records {
		if extractTopLevelDomain(req.Parameters.QName) == domain {
			// Define master server ips
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
	}

	return Response{Result: records}
}

// dnsQuery_GetDomainKeys retrieves DNSKEY records for a given domain.
func DnsQuery_GetDomainKeys(w http.ResponseWriter, r *http.Request, req Request) Response {
	for _, domain := range TLDRecords.records {
		if req.Parameters.QName == domain {
			keys := []struct {
				ID        int    `json:"id"`
				Flags     int    `json:"flags"`
				Active    bool   `json:"active"`
				Published bool   `json:"published"`
				Content   string `json:"content"`
			}{{
				ID:        3,
				Flags:     257,
				Active:    true,
				Published: true,
				Content:   domain + " IN DNSKEY 257 3 13 Ts7EglQbnyZDVklFGoiAnbB/DGzlJC4RBft7/wouiSxgQ9OB7sXD9yOkhyjhs5BzaOFs0LivpUwQZnYFkafAYA==",
			}}

			return Response{Result: keys}
		}
	}

	return Response{Result: nil}
}

func DnsQuery_List(w http.ResponseWriter, r *http.Request, req Request) Response {
	var records []config.DNSRecord
	var id int

	for key, domain := range TLDRecords.records {
		if extractTopLevelDomain(req.Parameters.Zonename) == domain {
			id = key
		}
	}

	for _, record := range StaticRecords.records {
		if extractTopLevelDomain(req.Parameters.Zonename) == extractTopLevelDomain(record.QName) {
			records = append(records, config.DNSRecord{
				DomainID: id,
				QName:    record.QName,
				QType:    record.QType,
				Content:  record.Content,
				TTL:      record.TTL,
				Auth:     true,
			})
		}
	}

	return Response{Result: records}
}
