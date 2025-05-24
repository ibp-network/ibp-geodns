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
type MonitorResultGeneric struct {
	MemberName string                 `json:"MemberName"`
	Status     bool                   `json:"Status"`
	ErrorText  string                 `json:"ErrorText"`
	Data       map[string]interface{} `json:"Data"`
}

var (
	officialResultsMu       sync.RWMutex
	officialResultsSnapshot OfficialResults
)

// UpdateOfficialResultsSnapshot replaces our local snapshot
func UpdateOfficialResultsSnapshot(newSnap OfficialResults) {
	officialResultsMu.Lock()
	defer officialResultsMu.Unlock()
	officialResultsSnapshot = newSnap
}

// GetOfficialResultsSnapshot safely returns a copy of the snapshot
func GetOfficialResultsSnapshot() OfficialResults {
	officialResultsMu.RLock()
	defer officialResultsMu.RUnlock()
	return officialResultsSnapshot
}
