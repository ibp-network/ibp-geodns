package api

import (
	"ibp-geodns/src/common/config"
	rpcMon "ibp-geodns/src/dnsApi/rpcMonitor"
	"sync"
)

// Define the StaticEntries type with a mutex
type StaticMap struct {
	mu      sync.RWMutex
	records []config.DNSRecord
}

type TLDMap struct {
	mu      sync.RWMutex
	records map[int]string
}

// ServicesMap wraps the global map of ServicesConfig with a mutex
type ServiceMap struct {
	mu       sync.RWMutex
	Services map[string]ServiceConfigs
}

// Services holds the service configuration and members for a domain
type ServiceConfigs struct {
	Name          string
	Active        int
	LevelRequired int
	NetworkName   string
	ServiceType   string
	Members       map[string]config.Member
}

// Request represents a DNS query request.
type Request struct {
	Method     string     `json:"method"`
	Parameters Parameters `json:"parameters"`
}

// Response represents a DNS query response.
type Response struct {
	Result interface{} `json:"result"`
}

// Parameters holds the parameters for DNS queries.
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
}

// DomainInfo provides information about a domain.
type DomainInfo struct {
	DomainID       int      `json:"id"`
	Zone           string   `json:"zone"`
	Masters        []string `json:"masters"`
	NotifiedSerial int      `json:"notified_serial"`
	Serial         int      `json:"serial"`
	LastCheck      int      `json:"last_check"`
	Kind           string   `json:"kind"`
}

type OfficialResults = struct {
	SiteResults     []rpcMon.SiteResult
	DomainResults   []rpcMon.DomainResult
	EndpointResults []rpcMon.EndpointResult
	Mu              sync.RWMutex
}
