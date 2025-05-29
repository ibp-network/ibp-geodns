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
			// For ANY, we return all matching records except ACME
			if (entry.QType == params.QType || params.QType == "ANY") &&
				(!strings.HasPrefix(domain, "_acme-challenge.")) {
				records = append(records, entry)
			}
		}
	}
	return records
}

// fetchACMEChallenge is used to retrieve ACME challenge content from a URL.
// Now includes a timeout and a simple size check for security.
func fetchACMEChallenge(url string) string {
	client := &http.Client{
		Timeout: 5 * time.Second, // added a short timeout for ACME retrieval
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Log(log.Error, "Failed to create ACME challenge request for %s: %v", url, err)
		return ""
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Log(log.Error, "Failed to fetch ACME challenge from %s: %v", url, err)
		return ""
	}
	defer resp.Body.Close()

	// Limit the amount of data we read to avoid huge memory usage.
	limitReader := io.LimitReader(resp.Body, 2048) // 2KB limit
	body, err := io.ReadAll(limitReader)
	if err != nil {
		log.Log(log.Error, "Failed to read ACME challenge body: %v", err)
		return ""
	}

	// Enforce a maximum length for the ACME content (e.g., 512 bytes).
	if len(body) > 512 {
		log.Log(log.Error, "ACME challenge data too large: length=%d from %s", len(body), url)
		return ""
	}

	return strings.TrimSpace(string(body))
}
