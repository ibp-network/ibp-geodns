package api

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
)

// StaticDNSEntries loads the c.StaticDNS into our StaticRecords global
func StaticDNSEntries() {
	c := cfg.GetConfig()
	StaticRecords.mu.Lock()
	defer StaticRecords.mu.Unlock()
	StaticRecords.records = c.StaticDNS
}

// ProcessSOA handles an SOA request
func ProcessSOA(params Parameters, id int, domain string) []cfg.DNSRecord {
	var records []cfg.DNSRecord
	tld := extractTopLevelDomain(domain)

	if params.QType == "SOA" {
		TLDRecords.mu.RLock()
		defer TLDRecords.mu.RUnlock()
		var domainFound int
		for key, storedDomain := range TLDRecords.records {
			if storedDomain == tld {
				domainFound = key
				break
			}
		}
		if domainFound != 0 {
			currentUnix := int(time.Now().UTC().Unix())
			records = append(records, cfg.DNSRecord{
				DomainID: id,
				QName:    tld,
				QType:    "SOA",
				Content:  fmt.Sprintf("dns-01.%s. hostmaster.%s. %d 3600 600 1209600 3600", tld, tld, currentUnix),
				TTL:      3600,
				Auth:     true,
			})
		}
	}
	return records
}

// ProcessACME handles potential ACME challenge
func ProcessACME(param Parameters, id int, domain string) []cfg.DNSRecord {
	var records []cfg.DNSRecord
	StaticRecords.mu.RLock()
	defer StaticRecords.mu.RUnlock()

	if strings.HasPrefix(domain, "_acme-challenge.") {
		for _, record := range StaticRecords.records {
			if record.QName == domain && record.QType == "TXT" {
				acmeContent := fetchACMEChallenge(record.Content)
				if acmeContent != "" {
					records = append(records, cfg.DNSRecord{
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
	return records
}

// ProcessNS handles NS queries
func ProcessNS(params Parameters, id int, domain string) []cfg.DNSRecord {
	var records []cfg.DNSRecord
	if params.QType == "NS" {
		StaticRecords.mu.RLock()
		defer StaticRecords.mu.RUnlock()
		for _, record := range StaticRecords.records {
			if record.QName == domain && record.QType == "NS" {
				records = append(records, record)
			}
		}
	}
	return records
}

// ProcessANY handles ANY queries
func ProcessANY(params Parameters, id int, domain string) []cfg.DNSRecord {
	var records []cfg.DNSRecord
	StaticRecords.mu.RLock()
	defer StaticRecords.mu.RUnlock()

	for _, entry := range StaticRecords.records {
		if entry.QName == domain {
			if (entry.QType == params.QType || params.QType == "ANY") &&
				(!strings.HasPrefix(domain, "_acme-challenge.")) {
				records = append(records, entry)
			}
		}
	}
	return records
}

// fetchACMEChallenge is used to retrieve ACME challenge content from a URL
func fetchACMEChallenge(url string) string {
	resp, err := http.Get(url)
	if err != nil {
		log.Log(log.Error, "Failed to fetch ACME challenge from %s: %v", url, err)
		return ""
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Log(log.Error, "Failed to read response body: %v", err)
		return ""
	}
	return strings.TrimSpace(string(body))
}
