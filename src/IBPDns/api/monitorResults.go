package api

import (
	"sync"
	"time"
)

// OfficialResults is our local copy of the monitor's official results
// after they reach consensus. This is used by the DNS API to serve queries.
type OfficialResults struct {
	SiteResults     []MonitorResultSite     `json:"SiteResults"`
	DomainResults   []MonitorResultDomain   `json:"DomainResults"`
	EndpointResults []MonitorResultEndpoint `json:"EndpointResults"`
}

// MonitorResultSite/Domain/Endpoint define the checks + results
type MonitorResultSite struct {
	CheckName string                 `json:"CheckName"`
	IsIPv6    bool                   `json:"IsIPv6"` // ADDED: Track IPv6 vs IPv4
	Results   []MonitorResultGeneric `json:"Results"`
}

type MonitorResultDomain struct {
	CheckName string                 `json:"CheckName"`
	Domain    string                 `json:"Domain"`
	IsIPv6    bool                   `json:"IsIPv6"` // ADDED: Track IPv6 vs IPv4
	Results   []MonitorResultGeneric `json:"Results"`
}

type MonitorResultEndpoint struct {
	CheckName string                 `json:"CheckName"`
	Domain    string                 `json:"Domain"`
	RpcUrl    string                 `json:"RpcUrl"`
	IsIPv6    bool                   `json:"IsIPv6"` // ADDED: Track IPv6 vs IPv4
	Results   []MonitorResultGeneric `json:"Results"`
}

type MonitorResultGeneric struct {
	MemberName string                 `json:"MemberName"`
	Status     bool                   `json:"Status"`
	ErrorText  string                 `json:"ErrorText"`
	Data       map[string]interface{} `json:"Data"`
	IsIPv6     bool                   `json:"IsIPv6"`
	Checktime  time.Time              `json:"Checktime"` // ← NEW
}

// We rename the underlying variables to reflect they are official results
var (
	officialResultsMu       sync.RWMutex
	officialResultsSnapshot OfficialResults
)

// SetOfficialSnapshot updates our local memory copy of the official results
func SetOfficialSnapshot(newSnap OfficialResults) {
	officialResultsMu.Lock()
	defer officialResultsMu.Unlock()
	officialResultsSnapshot = newSnap
}

// GetOfficialSnapshot returns a copy of the official results snapshot
func GetOfficialSnapshot() OfficialResults {
	officialResultsMu.RLock()
	defer officialResultsMu.RUnlock()
	// For safety, do a shallow copy by value
	return officialResultsSnapshot
}
