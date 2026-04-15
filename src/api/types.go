package api

import (
	"sync"

	cfg "github.com/ibp-network/ibp-geodns-libs/config"
)

type StaticMap struct {
	mu      sync.RWMutex
	records []cfg.DNSRecord
}

var StaticRecords = &StaticMap{
	records: []cfg.DNSRecord{},
}

type TLDMap struct {
	mu      sync.RWMutex
	records map[int]string
	ids     map[string]int
}

var TLDRecords = &TLDMap{
	records: make(map[int]string),
	ids:     make(map[string]int),
}

type ServiceMap struct {
	mu       sync.RWMutex
	Services map[string]ServiceConfigs
}

var ServiceRecords = &ServiceMap{
	Services: make(map[string]ServiceConfigs),
}

type ServiceConfigs struct {
	Name          string
	Active        int
	LevelRequired int
	NetworkName   string
	ServiceType   string
	Members       map[string]cfg.Member
}

type Request struct {
	Method     string     `json:"method"`
	Parameters Parameters `json:"parameters"`
}
type Response struct {
	Result interface{} `json:"result"`
}

type Parameters struct {
	DomainID   string `json:"domain_id"`
	Local      string `json:"local"`
	Zonename   string `json:"zonename"`
	QName      string `json:"qname"`
	QType      string `json:"qtype"`
	RealRemote string `json:"real-remote"`
	Remote     string `json:"remote"`
	ZoneID     int    `json:"zone-id"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Key        struct {
		ID        int    `json:"id"`
		Flags     int    `json:"flags"`
		Active    bool   `json:"active"`
		Published bool   `json:"published"`
		Content   string `json:"content"`
	} `json:"key"`

	MemberName string `json:"memberName"`
	Domain     string `json:"domain"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
}

type DomainInfo struct {
	DomainID       int      `json:"id"`
	Zone           string   `json:"zone"`
	Masters        []string `json:"masters"`
	NotifiedSerial int      `json:"notified_serial"`
	Serial         int      `json:"serial"`
	LastCheck      int      `json:"last_check"`
	Kind           string   `json:"kind"`
}

// CountryOverride represents a country code override mapping
type CountryOverride struct {
	// MemberName is the name of the member to route to (takes precedence over IPs)
	MemberName string `json:"memberName,omitempty"`
	// IPv4 is the IPv4 address to route to (used if MemberName is empty)
	IPv4 string `json:"ipv4,omitempty"`
	// IPv6 is the IPv6 address to route to (used if MemberName is empty)
	IPv6 string `json:"ipv6,omitempty"`
}

// CountryOverrideMap stores country code overrides per domain
type CountryOverrideMap struct {
	mu        sync.RWMutex
	overrides map[string]map[string]CountryOverride // domain -> country code -> override
}

var CountryOverrides = &CountryOverrideMap{
	overrides: make(map[string]map[string]CountryOverride),
}
