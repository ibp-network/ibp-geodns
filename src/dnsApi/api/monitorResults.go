package api

import (
	"sync"
)

// OfficialResults is our local copy of the serviceMonitor's official results
type OfficialResults struct {
	SiteResults     []MonitorResultSite     `json:"SiteResults"`
	DomainResults   []MonitorResultDomain   `json:"DomainResults"`
	EndpointResults []MonitorResultEndpoint `json:"EndpointResults"`
}

// MonitorResultSite/Domain/Endpoint define the checks + results
type MonitorResultSite struct {
	CheckName string                 `json:"CheckName"`
	Results   []MonitorResultGeneric `json:"Results"`
}
type MonitorResultDomain struct {
	CheckName string                 `json:"CheckName"`
	Domain    string                 `json:"Domain"`
	Results   []MonitorResultGeneric `json:"Results"`
}
type MonitorResultEndpoint struct {
	CheckName string                 `json:"CheckName"`
	Domain    string                 `json:"Domain"`
	RpcUrl    string                 `json:"RpcUrl"`
	Results   []MonitorResultGeneric `json:"Results"`
}

// MonitorResultGeneric holds each member result
type MonitorResultGeneric struct {
	MemberName string                 `json:"MemberName"`
	Status     bool                   `json:"Status"`
	ErrorText  string                 `json:"ErrorText"`
	Data       map[string]interface{} `json:"Data"`
}

// Local snapshot + mutex
var (
	dnsMonitorMu       sync.RWMutex
	dnsMonitorSnapshot OfficialResults
)

// SetLocalSnapshot updates our local memory copy of the official results
func SetLocalSnapshot(newSnap OfficialResults) {
	dnsMonitorMu.Lock()
	defer dnsMonitorMu.Unlock()
	dnsMonitorSnapshot = newSnap
}

// GetLocalSnapshot returns a copy of the local snapshot
func GetLocalSnapshot() OfficialResults {
	dnsMonitorMu.RLock()
	defer dnsMonitorMu.RUnlock()
	// You could do a deep copy if you prefer. For now, just return by value.
	return dnsMonitorSnapshot
}
