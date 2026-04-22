package api

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	cfg "github.com/ibp-network/ibp-geodns-libs/config"
)

func resetTestState() {
	StaticRecords.mu.Lock()
	StaticRecords.records = nil
	StaticRecords.mu.Unlock()

	ServiceRecords.mu.Lock()
	ServiceRecords.Services = make(map[string]ServiceConfigs)
	ServiceRecords.mu.Unlock()

	TLDRecords.mu.Lock()
	TLDRecords.records = make(map[int]string)
	TLDRecords.ids = make(map[string]int)
	TLDRecords.mu.Unlock()

	CountryOverrides.mu.Lock()
	CountryOverrides.overrides = make(map[string]map[string]CountryOverride)
	CountryOverrides.mu.Unlock()

	ClearOfficialSnapshot()
	SetConfigPath("ibpdns.json")
}

func setStaticRecords(records []cfg.DNSRecord) {
	StaticRecords.mu.Lock()
	StaticRecords.records = append([]cfg.DNSRecord(nil), records...)
	StaticRecords.mu.Unlock()
	populateTLDRecords()
}

func setServiceRecords(records map[string]ServiceConfigs) {
	ServiceRecords.mu.Lock()
	ServiceRecords.Services = records
	ServiceRecords.mu.Unlock()
	populateTLDRecords()
}

func makeMember(name, ipv4, ipv6 string, lat, lon float64) cfg.Member {
	return cfg.Member{
		Details: cfg.MemberDetails{
			Name: name,
		},
		Service: cfg.ServiceInfo{
			Active:      1,
			ServiceIPv4: ipv4,
			ServiceIPv6: ipv6,
		},
		Location: cfg.Location{
			Latitude:  lat,
			Longitude: lon,
		},
	}
}

func mustLookupTLDID(t *testing.T, domain string) int {
	t.Helper()
	id, ok := lookupTLDID(domain)
	if !ok {
		t.Fatalf("expected TLD id for %s", domain)
	}
	return id
}

func decodeJSONResult[T any](t *testing.T, value interface{}) T {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}

	var decoded T
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	return decoded
}

func TestHandleDNSQueryUsesRealRemoteBeforeRemoteAndParsesHostPort(t *testing.T) {
	resetTestState()
	setServiceRecords(map[string]ServiceConfigs{
		"rpc.example.com": {
			Active: 1,
			Members: map[string]cfg.Member{
				"a-far":  makeMember("a-far", "192.0.2.20", "", 50, 50),
				"b-near": makeMember("b-near", "192.0.2.10", "", 1, 1),
			},
		},
	})

	res := handle_DNSQuery(Request{
		Method: "lookup",
		Parameters: Parameters{
			QName:      "rpc.example.com.",
			QType:      "A",
			Remote:     "not-an-ip",
			RealRemote: "203.0.113.1:5300",
		},
	})

	records, ok := res.Result.([]cfg.DNSRecord)
	if !ok {
		t.Fatalf("expected DNS records, got %T", res.Result)
	}
	if len(records) != 1 {
		t.Fatalf("expected one record, got %d", len(records))
	}
	if records[0].Content != "192.0.2.10" {
		t.Fatalf("expected nearest member to be selected, got %s", records[0].Content)
	}
}

func TestProcessCountryOverrideDoesNotFallThroughToDirectIPWhenMemberOverrideFails(t *testing.T) {
	resetTestState()
	setServiceRecords(map[string]ServiceConfigs{
		"example.com": {
			Active: 1,
			Members: map[string]cfg.Member{
				"member-a": makeMember("member-a", "192.0.2.10", "", 1, 1),
			},
		},
	})

	override := CountryOverride{
		MemberName: "missing-member",
		IPv4:       "203.0.113.10",
	}
	records, chosen := processCountryOverride(
		Parameters{},
		101,
		"example.com",
		false,
		override,
		"US",
		"203.0.113.1",
	)
	if len(records) != 0 || chosen != "" {
		t.Fatalf("expected geographic fallback, got records=%v chosen=%q", records, chosen)
	}
}

func TestHandleNatsCountryOverrideUpdateClearNormalizesDomain(t *testing.T) {
	resetTestState()
	CountryOverrides.SetCountryOverride("example.com", "US", CountryOverride{IPv4: "203.0.113.10"})

	handleNatsCountryOverrideUpdate([]byte(`{"action":"clear","domain":"Example.COM"}`))

	if _, ok := CountryOverrides.GetCountryOverride("example.com", "US"); ok {
		t.Fatal("expected normalized clear to remove the override")
	}
}

func TestLoadCountryOverridesFromConfigFileUsesConfiguredPath(t *testing.T) {
	resetTestState()

	configPath := filepath.Join(t.TempDir(), "custom-overrides.json")
	configJSON := `{"GeoDNSOverrides":{"Example.COM":{"us":{"ipv4":"203.0.113.10"}}}}`
	if err := os.WriteFile(configPath, []byte(configJSON), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	SetConfigPath(configPath)
	LoadCountryOverridesFromConfigFile()

	override, ok := CountryOverrides.GetCountryOverride("example.com", "US")
	if !ok {
		t.Fatal("expected override to be loaded from configured path")
	}
	if override.IPv4 != "203.0.113.10" {
		t.Fatalf("expected IPv4 override to be loaded, got %q", override.IPv4)
	}
}

func TestMemberOnlineChecksUseCaseInsensitiveMatchesAndExplicitSnapshotPolicy(t *testing.T) {
	resetTestState()

	if !IsMemberOnlineForDomainIPv4("example.com", "member-a") {
		t.Fatal("expected missing snapshot to keep permissive routing")
	}

	SetOfficialSnapshot(OfficialResults{})
	if !IsMemberOnlineForDomainIPv4("example.com", "member-a") {
		t.Fatal("expected empty snapshot to keep permissive routing")
	}

	SetOfficialSnapshot(OfficialResults{
		DomainResults: []MonitorResultDomain{{
			Domain: "example.com",
			IsIPv6: false,
			Results: []MonitorResultGeneric{{
				MemberName: "MEMBER-A",
				Status:     false,
			}},
		}},
	})

	if IsMemberOnlineForDomainIPv4("example.com", "member-a") {
		t.Fatal("expected case-insensitive offline match to mark member offline")
	}
}

func TestHandleGetDomainInfoUsesNameAndReturnsSingleObject(t *testing.T) {
	resetTestState()
	setStaticRecords([]cfg.DNSRecord{
		{QName: "Example.COM.", QType: "NS", Content: "dns-01.example.com.", TTL: 3600},
		{QName: "DNS-01.Example.com.", QType: "A", Content: "198.51.100.1", TTL: 3600},
	})

	res := handle_GetDomainInfo(Request{
		Parameters: Parameters{
			Name: "Example.com.",
		},
	})

	info, ok := res.Result.(DomainInfo)
	if !ok {
		t.Fatalf("expected DomainInfo result, got %T", res.Result)
	}
	if info.Zone != "example.com" {
		t.Fatalf("expected normalized zone, got %q", info.Zone)
	}
	if len(info.Masters) != 1 || info.Masters[0] != "198.51.100.1" {
		t.Fatalf("expected master record to be derived from static DNS entries, got %v", info.Masters)
	}
}

func TestHandleGetAllDomainMetadataReturnsMetadataMapForZone(t *testing.T) {
	resetTestState()
	setStaticRecords([]cfg.DNSRecord{
		{QName: "example.com", QType: "NS", Content: "dns-01.example.com", TTL: 3600},
	})

	res := handle_GetAllDomainMetadata(Request{
		Parameters: Parameters{
			Name: "Example.com.",
		},
	})

	metadata, ok := res.Result.(map[string][]string)
	if !ok {
		t.Fatalf("expected metadata map, got %T", res.Result)
	}
	if got := metadata["PRESIGNED"]; len(got) != 1 || got[0] != "0" {
		t.Fatalf("expected PRESIGNED metadata, got %#v", metadata)
	}
}

func TestHandleGetDomainKeysUsesNameAndCaseInsensitiveZoneLookup(t *testing.T) {
	resetTestState()
	setStaticRecords([]cfg.DNSRecord{
		{QName: "example.com", QType: "NS", Content: "dns-01.example.com", TTL: 3600},
	})

	res := handle_GetDomainKeys(Request{
		Parameters: Parameters{
			Name: "Example.com.",
		},
	})

	keys := decodeJSONResult[[]map[string]interface{}](t, res.Result)
	if len(keys) != 0 {
		t.Fatalf("expected no DNSSEC keys until real signing is configured, got %#v", keys)
	}
}

func TestHandleDNSQueryANYNormalizesStaticResponseRecords(t *testing.T) {
	resetTestState()
	setStaticRecords([]cfg.DNSRecord{
		{QName: "Example.COM.", QType: "NS", Content: "dns-01.example.com.", TTL: 3600, DomainID: 999},
		{QName: "EXAMPLE.com", QType: "TXT", Content: "hello", TTL: 60, DomainID: 123},
	})

	expectedID := mustLookupTLDID(t, "example.com")
	res := handle_DNSQuery(Request{
		Parameters: Parameters{
			QName: "example.com.",
			QType: "ANY",
		},
	})

	records, ok := res.Result.([]cfg.DNSRecord)
	if !ok {
		t.Fatalf("expected DNS records, got %T", res.Result)
	}
	if len(records) != 2 {
		t.Fatalf("expected two static records, got %d", len(records))
	}
	for _, record := range records {
		if record.DomainID != expectedID {
			t.Fatalf("expected normalized domain id %d, got %d", expectedID, record.DomainID)
		}
		if !record.Auth {
			t.Fatalf("expected auth=true for %#v", record)
		}
		if record.QName != "example.com" {
			t.Fatalf("expected normalized qname, got %q", record.QName)
		}
	}
}

func TestHandleDNSQueryNSNormalizesResponseRecords(t *testing.T) {
	resetTestState()
	setStaticRecords([]cfg.DNSRecord{
		{QName: "Example.COM.", QType: "NS", Content: "dns-01.example.com.", TTL: 3600, DomainID: 999},
	})

	expectedID := mustLookupTLDID(t, "example.com")
	res := handle_DNSQuery(Request{
		Parameters: Parameters{
			QName: "example.com.",
			QType: "NS",
		},
	})

	records, ok := res.Result.([]cfg.DNSRecord)
	if !ok {
		t.Fatalf("expected DNS records, got %T", res.Result)
	}
	if len(records) != 1 {
		t.Fatalf("expected one NS record, got %d", len(records))
	}
	if records[0].DomainID != expectedID {
		t.Fatalf("expected normalized domain id %d, got %d", expectedID, records[0].DomainID)
	}
	if !records[0].Auth || records[0].QName != "example.com" {
		t.Fatalf("expected normalized NS response, got %#v", records[0])
	}
}

func TestHandleDNSQueryNormalizesLowercaseQType(t *testing.T) {
	resetTestState()
	setStaticRecords([]cfg.DNSRecord{
		{QName: "Example.COM.", QType: "NS", Content: "dns-01.example.com.", TTL: 3600},
	})

	res := handle_DNSQuery(Request{
		Parameters: Parameters{
			QName: "example.com.",
			QType: "ns",
		},
	})

	records, ok := res.Result.([]cfg.DNSRecord)
	if !ok {
		t.Fatalf("expected DNS records, got %T", res.Result)
	}
	if len(records) != 1 {
		t.Fatalf("expected one NS record for lowercase qtype, got %d", len(records))
	}
	if records[0].QType != "NS" {
		t.Fatalf("expected NS record, got %#v", records[0])
	}
}

func TestHandleGetDomainListSynthesizesAuthorityRecordsForDynamicZones(t *testing.T) {
	resetTestState()
	setServiceRecords(map[string]ServiceConfigs{
		"rpc.example.com": {
			Active: 1,
		},
	})

	expectedID := mustLookupTLDID(t, "rpc.example.com")
	res := handle_GetDomainList(Request{
		Parameters: Parameters{
			Zonename: "rpc.example.com",
		},
	})

	records, ok := res.Result.([]cfg.DNSRecord)
	if !ok {
		t.Fatalf("expected DNS records, got %T", res.Result)
	}
	if len(records) < 2 {
		t.Fatalf("expected synthesized SOA and NS records, got %#v", records)
	}

	var sawSOA, sawNS bool
	for _, record := range records {
		if record.DomainID != expectedID {
			t.Fatalf("expected normalized domain id %d, got %d", expectedID, record.DomainID)
		}
		if record.QName != "rpc.example.com" || !record.Auth {
			t.Fatalf("expected normalized authority record, got %#v", record)
		}
		if record.QType == "SOA" {
			sawSOA = true
		}
		if record.QType == "NS" {
			sawNS = true
		}
	}
	if !sawSOA || !sawNS {
		t.Fatalf("expected both SOA and NS records, got %#v", records)
	}
}

func TestHandleGetDomainListAddsAuthorityRecordsWhenDynamicZoneHasStaticRecords(t *testing.T) {
	resetTestState()
	setStaticRecords([]cfg.DNSRecord{
		{QName: "rpc.example.com", QType: "TXT", Content: "hello", TTL: 60},
	})
	setServiceRecords(map[string]ServiceConfigs{
		"rpc.example.com": {
			Active: 1,
		},
	})

	expectedID := mustLookupTLDID(t, "rpc.example.com")
	res := handle_GetDomainList(Request{
		Parameters: Parameters{
			Zonename: "rpc.example.com",
		},
	})

	records, ok := res.Result.([]cfg.DNSRecord)
	if !ok {
		t.Fatalf("expected DNS records, got %T", res.Result)
	}
	if len(records) < 3 {
		t.Fatalf("expected static record plus synthesized authority records, got %#v", records)
	}

	var sawTXT, sawSOA, sawNS bool
	for _, record := range records {
		if record.DomainID != expectedID {
			t.Fatalf("expected normalized domain id %d, got %d", expectedID, record.DomainID)
		}
		if record.QName != "rpc.example.com" || !record.Auth {
			t.Fatalf("expected normalized authority record, got %#v", record)
		}
		switch record.QType {
		case "TXT":
			sawTXT = sawTXT || record.Content == "hello"
		case "SOA":
			sawSOA = true
		case "NS":
			sawNS = true
		}
	}
	if !sawTXT || !sawSOA || !sawNS {
		t.Fatalf("expected TXT, SOA, and NS records, got %#v", records)
	}
}
