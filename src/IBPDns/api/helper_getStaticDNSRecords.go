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

// fetchACMEChallenge retrieves ACME challenge content from a URL, with up to 3 short retries.
func fetchACMEChallenge(url string) string {
	var finalBody string

	for attempt := 1; attempt <= 3; attempt++ {
		body, err := tryFetchACMEOnce(url)
		if err == nil && body != "" {
			finalBody = body
			break
		}
		// If we have not succeeded, wait briefly before retrying
		time.Sleep(1 * time.Second)
		log.Log(log.Warn, "fetchACMEChallenge attempt %d failed for %s: %v", attempt, url, err)
	}

	return finalBody
}

// tryFetchACMEOnce does a single GET request to the ACME challenge URL with a short timeout.
func tryFetchACMEOnce(url string) (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Second, // short timeout for ACME retrieval
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Limit the amount of data we read to avoid huge memory usage.
	limitReader := io.LimitReader(resp.Body, 2048) // 2KB limit
	body, err := io.ReadAll(limitReader)
	if err != nil {
		return "", err
	}

	// Enforce a maximum length for the ACME content (e.g., 512 bytes).
	if len(body) > 512 {
		return "", fmt.Errorf("ACME challenge data too large (length=%d)", len(body))
	}

	return strings.TrimSpace(string(body)), nil
}
