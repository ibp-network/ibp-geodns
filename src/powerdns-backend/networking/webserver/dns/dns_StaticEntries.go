package dns

import (
	"fmt"
	"ibp-geodns/config"
	l "ibp-geodns/logging"
	"io"
	"net/http"
	"strings"
	"time"
)

// StaticDNSEntries synchronizes staticEntries with c.StaticDNS
func StaticDNSEntries() {
	c := config.GetConfig()

	StaticRecords.mu.Lock()
	defer StaticRecords.mu.Unlock()
	StaticRecords.records = c.StaticDNS
}

func ProcessSOA(params Parameters, id int, domain string) []config.DNSRecord {
	var records []config.DNSRecord

	tld := extractTopLevelDomain(domain)

	// Handle SOA queries
	if params.QType == "SOA" {
		// Lock mutex before accessing topLevelDomains
		domainFound := 0
		for key, storedDomain := range TLDRecords.records {
			if storedDomain == tld {
				domainFound = key
				break
			}
		}

		if domainFound != 0 {
			currentUnixTimestamp := int(time.Now().UTC().Unix())

			// Insert SOA return record
			records = append(records, config.DNSRecord{
				DomainID: id,
				QName:    tld,
				QType:    "SOA",
				Content:  fmt.Sprintf("dns-01.%s. hostmaster.%s. %d 3600 600 1209600 3600", tld, tld, currentUnixTimestamp),
				TTL:      3600,
				Auth:     true,
			})
		}
	}

	return records
}

func ProcessACME(param Parameters, id int, domain string) []config.DNSRecord {
	var records []config.DNSRecord

	StaticRecords.mu.RLock()
	defer StaticRecords.mu.RUnlock()

	// Check for ACME challenge records
	if strings.HasPrefix(domain, "_acme-challenge.") {
		for _, record := range StaticRecords.records {
			if record.QName == domain {
				if record.QType == "TXT" {
					acmeContent := fetchACMEChallenge(record.Content)
					if acmeContent != "" {
						// Insert response record for ACME
						records = append(records, config.DNSRecord{
							DomainID: id,
							QName:    record.QName,
							QType:    "TXT",
							Content:  acmeContent,
							TTL:      record.TTL,
							Auth:     true,
						})
					}
				}
			}
		}
	}

	return records
}

func ProcessNS(params Parameters, id int, domain string) []config.DNSRecord {
	var records []config.DNSRecord

	StaticRecords.mu.RLock()
	defer StaticRecords.mu.RUnlock()

	// Handle NS queries
	if params.QType == "NS" {
		for _, record := range StaticRecords.records {
			if record.QName == domain && record.QType == "NS" {
				records = append(records, record)
			}
		}
	}

	return records
}

func ProcessANY(params Parameters, id int, domain string) []config.DNSRecord {
	var records []config.DNSRecord

	StaticRecords.mu.RLock()
	defer StaticRecords.mu.RUnlock()

	for _, entry := range StaticRecords.records {
		if entry.QName == domain {
			if (entry.QType == params.QType || params.QType == "ANY") && (!strings.HasPrefix(domain, "_acme-challenge.")) {
				// Append all static entries to records variable
				records = append(records, entry)
			}
		}
	}

	return records
}

// fetchACMEChallenge retrieves the ACME challenge content from the specified URL.
func fetchACMEChallenge(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		l.Log(l.Error, "DNSLookup: failed to fetch ACME challenge from %s: %+v", url, err)
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		l.Log(l.Error, "DNSLookup: failed to read response body: %+v", err)
		return ""
	}

	content := strings.TrimSpace(string(body))
	return content
}
