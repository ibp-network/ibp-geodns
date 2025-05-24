package monitor

import (
	"sync"
	"time"

	cfg "ibp-geodns/src/common/config"
)

var (
	officialMu sync.RWMutex
	official   = OfficialResults{}
)

type OfficialResults struct {
	SiteResults     []SiteResult     `json:"SiteResults"`
	DomainResults   []DomainResult   `json:"DomainResults"`
	EndpointResults []EndpointResult `json:"EndpointResults"`
}

type SiteResult struct {
	Check   cfg.Check `json:"Check"`
	Results []Result  `json:"Results"`
}

type DomainResult struct {
	Check   cfg.Check   `json:"Check"`
	Service cfg.Service `json:"Service"`
	Domain  string      `json:"Domain"`
	Results []Result    `json:"Results"`
}

type EndpointResult struct {
	Check    cfg.Check   `json:"Check"`
	Service  cfg.Service `json:"Service"`
	RpcUrl   string      `json:"RpcUrl"`
	Protocol string      `json:"Protocol"`
	Domain   string      `json:"Domain"`
	Port     string      `json:"Port"`
	Path     string      `json:"Path"`
	Results  []Result    `json:"Results"`
}

type Result struct {
	Member    cfg.Member             `json:"Member"`
	Status    bool                   `json:"Status"`
	Checktime time.Time              `json:"Checktime"`
	ErrorText string                 `json:"ErrorText"`
	Data      map[string]interface{} `json:"Data"`
}

// GetOfficialResults returns the current official results in a snapshot
func GetOfficialResults() OfficialResults {
	officialMu.RLock()
	defer officialMu.RUnlock()
	return official
}

// ResetOfficialResults clears them
func ResetOfficialResults() {
	officialMu.Lock()
	defer officialMu.Unlock()
	official = OfficialResults{}
}

// UpdateSiteResults updates the official site results
func SetOfficialSiteResults(results []SiteResult) {
	officialMu.Lock()
	defer officialMu.Unlock()
	official.SiteResults = results
}

// UpdateDomainResults updates official domain
func SetOfficialDomainResults(results []DomainResult) {
	officialMu.Lock()
	defer officialMu.Unlock()
	official.DomainResults = results
}

// UpdateEndpointResults updates official endpoint
func SetOfficialEndpointResults(results []EndpointResult) {
	officialMu.Lock()
	defer officialMu.Unlock()
	official.EndpointResults = results
}
