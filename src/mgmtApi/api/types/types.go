package types

import (
	cfg "ibp-geodns/src/common/config"
	mon "ibp-geodns/src/dnsApi/monitor"
	"sync"
)

// Define the StaticEntries type with a mutex
type StaticMap struct {
	mu      sync.RWMutex
	records []cfg.DNSRecord
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
	Members       map[string]cfg.Member
}

// Request represents a DNS query request.
type Request struct {
	Method     string     `json:"method"`
	Parameters Parameters `json:"parameters"`
}

// Response represents a DNS query response.
type Response struct {
	Result interface{} `json:"result"`
	Error  string      `json:"Error"`
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
	SiteResults     []mon.SiteResult
	DomainResults   []mon.DomainResult
	EndpointResults []mon.EndpointResult
	Mu              sync.RWMutex
}

// Response represents a DNS query response.
type ApiResponse struct {
	Result interface{} `json:"result"`
	Error  string      `json:"error"`
}

// ApiRequest represents an API request structure.
type ApiRequest struct {
	Method     string `json:"Method"`
	Action     string `json:"Action"`
	Output     string `json:"Output"`
	MemberName string `json:"MemberName"`
	Year       string `json:"Year"`
	Month      string `json:"Month"`
	StartYear  string `json:"StartYear"`
	StartMonth string `json:"StartMonth"`
	EndYear    string `json:"EndYear"`
	EndMonth   string `json:"EndMonth"`
	AuthKey    string `json:"Authkey"`
}

type CheckResult struct {
	CheckName string                 `json:"checkName"`
	Status    bool                   `json:"status"`
	ErrorText string                 `json:"errorText"`
	Data      map[string]interface{} `json:"data"`
}
