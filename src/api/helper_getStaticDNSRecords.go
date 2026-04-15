package api

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	cfg "github.com/ibp-network/ibp-geodns-libs/config"
	log "github.com/ibp-network/ibp-geodns-libs/logging"
)

func StaticDNSEntries() {
	c := cfg.GetConfig()
	StaticRecords.mu.Lock()
	defer StaticRecords.mu.Unlock()
	StaticRecords.records = c.StaticDNS
}

func ProcessSOA(params Parameters, id int, domain string) []cfg.DNSRecord {
	var records []cfg.DNSRecord
	tld := normalizeDomain(domain)

	if params.QType == "SOA" {
		if resolvedID, ok := lookupTLDID(tld); ok {
			id = resolvedID
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

func ProcessACME(param Parameters, id int, domain string) []cfg.DNSRecord {
	var records []cfg.DNSRecord
	StaticRecords.mu.RLock()
	defer StaticRecords.mu.RUnlock()

	if strings.HasPrefix(domain, "_acme-challenge.") {
		for _, record := range StaticRecords.records {
			if normalizeDomain(record.QName) == domain && record.QType == "TXT" {
				acmeContent := fetchACMEChallenge(record.Content)
				if acmeContent != "" {
					records = append(records, cfg.DNSRecord{
						DomainID: id,
						QName:    domain,
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

func ProcessNS(params Parameters, id int, domain string) []cfg.DNSRecord {
	var records []cfg.DNSRecord
	if params.QType == "NS" {
		StaticRecords.mu.RLock()
		defer StaticRecords.mu.RUnlock()
		for _, record := range StaticRecords.records {
			if normalizeDomain(record.QName) == domain && record.QType == "NS" {
				records = append(records, recordForResponse(record, id, domain))
			}
		}
	}
	return records
}

func ProcessANY(params Parameters, id int, domain string) []cfg.DNSRecord {
	var records []cfg.DNSRecord
	StaticRecords.mu.RLock()
	defer StaticRecords.mu.RUnlock()

	for _, entry := range StaticRecords.records {
		if normalizeDomain(entry.QName) == domain {
			if (entry.QType == params.QType || params.QType == "ANY") &&
				(!strings.HasPrefix(domain, "_acme-challenge.")) {
				records = append(records, recordForResponse(entry, id, domain))
			}
		}
	}
	return records
}

func fetchACMEChallenge(url string) string {
	var finalBody string

	for attempt := 1; attempt <= 3; attempt++ {
		body, err := tryFetchACMEOnce(url)
		if err == nil && body != "" {
			finalBody = body
			break
		}
		time.Sleep(1 * time.Second)
		log.Log(log.Warn, "fetchACMEChallenge attempt %d failed for %s: %v", attempt, url, err)
	}

	return finalBody
}

func tryFetchACMEOnce(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("unsupported ACME URL scheme: %s", parsedURL.Scheme)
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}
	req, err := http.NewRequest("GET", parsedURL.String(), nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected ACME status code: %d", resp.StatusCode)
	}

	limitReader := io.LimitReader(resp.Body, 2048)
	body, err := io.ReadAll(limitReader)
	if err != nil {
		return "", err
	}

	if len(body) > 512 {
		return "", fmt.Errorf("ACME challenge data too large (length=%d)", len(body))
	}

	return strings.TrimSpace(string(body)), nil
}
