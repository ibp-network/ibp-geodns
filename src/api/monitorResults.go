package api

import (
	"sync"
	"time"
)

type OfficialResults struct {
	SiteResults     []MonitorResultSite     `json:"SiteResults"`
	DomainResults   []MonitorResultDomain   `json:"DomainResults"`
	EndpointResults []MonitorResultEndpoint `json:"EndpointResults"`
}

type MonitorResultSite struct {
	CheckName string                 `json:"CheckName"`
	IsIPv6    bool                   `json:"IsIPv6"`
	Results   []MonitorResultGeneric `json:"Results"`
}

type MonitorResultDomain struct {
	CheckName string                 `json:"CheckName"`
	Domain    string                 `json:"Domain"`
	IsIPv6    bool                   `json:"IsIPv6"`
	Results   []MonitorResultGeneric `json:"Results"`
}

type MonitorResultEndpoint struct {
	CheckName string                 `json:"CheckName"`
	Domain    string                 `json:"Domain"`
	RpcUrl    string                 `json:"RpcUrl"`
	IsIPv6    bool                   `json:"IsIPv6"`
	Results   []MonitorResultGeneric `json:"Results"`
}

type MonitorResultGeneric struct {
	MemberName string                 `json:"MemberName"`
	Status     bool                   `json:"Status"`
	ErrorText  string                 `json:"ErrorText"`
	Data       map[string]interface{} `json:"Data"`
	IsIPv6     bool                   `json:"IsIPv6"`
	Checktime  time.Time              `json:"Checktime"`
}

var (
	officialResultsMu       sync.RWMutex
	officialResultsSnapshot OfficialResults
	officialResultsLoaded   bool
)

func SetOfficialSnapshot(newSnap OfficialResults) {
	officialResultsMu.Lock()
	defer officialResultsMu.Unlock()
	officialResultsSnapshot = newSnap
	officialResultsLoaded = true
}

func GetOfficialSnapshot() OfficialResults {
	officialResultsMu.RLock()
	defer officialResultsMu.RUnlock()
	return officialResultsSnapshot
}

func ClearOfficialSnapshot() {
	officialResultsMu.Lock()
	defer officialResultsMu.Unlock()
	officialResultsSnapshot = OfficialResults{}
	officialResultsLoaded = false
}

func HasOfficialSnapshot() bool {
	officialResultsMu.RLock()
	defer officialResultsMu.RUnlock()
	return officialResultsLoaded
}

func HasOfficialMonitorData(snap OfficialResults) bool {
	return len(snap.SiteResults) > 0 || len(snap.DomainResults) > 0 || len(snap.EndpointResults) > 0
}
