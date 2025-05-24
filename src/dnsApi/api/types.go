package api

import (
	"sync"

	cfg "ibp-geodns/src/common/config"
)

// StaticMap holds a list of static DNS records with a mutex
type StaticMap struct {
	mu      sync.RWMutex
	records []cfg.DNSRecord
}

// Global static records container
var StaticRecords = &StaticMap{
	records: []cfg.DNSRecord{},
}

// TLDMap holds TLDs with a mutex
type TLDMap struct {
	mu      sync.RWMutex
	records map[int]string
}

// Global TLD map
var TLDRecords = &TLDMap{
	records: make(map[int]string),
}

// ServiceMap holds dynamic service configs with a mutex
type ServiceMap struct {
	mu       sync.RWMutex
	Services map[string]ServiceConfigs
}

// Global service map
var ServiceRecords = &ServiceMap{
	Services: make(map[string]ServiceConfigs),
}

// ServiceConfigs holds info about a particular service domain and its members
type ServiceConfigs struct {
	Name          string
	Active        int
	LevelRequired int
	NetworkName   string
	ServiceType   string
	Members       map[string]cfg.Member
}

// Request and Response are used for the DNS API
type Request struct {
	Method     string     `json:"method"`
	Parameters Parameters `json:"parameters"`
}
type Response struct {
	Result interface{} `json:"result"`
}

// Parameters for DNS queries
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

// DomainInfo is a helper struct used in some handlers
type DomainInfo struct {
	DomainID       int      `json:"id"`
	Zone           string   `json:"zone"`
	Masters        []string `json:"masters"`
	NotifiedSerial int      `json:"notified_serial"`
	Serial         int      `json:"serial"`
	LastCheck      int      `json:"last_check"`
	Kind           string   `json:"kind"`
}
