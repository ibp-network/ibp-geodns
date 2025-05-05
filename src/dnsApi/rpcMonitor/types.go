package rpcMonitor

import (
	"sync"
	"time"

	cfg "ibp-geodns/src/common/config"
)

// Results type
type Results struct {
	SiteResults     []SiteResult
	DomainResults   []DomainResult
	EndpointResults []EndpointResult
	Mu              sync.RWMutex
}

// SiteResult represents the result of a site check
type SiteResult struct {
	Check   cfg.Check
	Results []Result
	Mu      sync.RWMutex
}

// DomainResult represents the result of a domain check
type DomainResult struct {
	Check   cfg.Check
	Service cfg.Service
	Domain  string
	Results []Result
	Mu      sync.RWMutex
}

// EndpointResult represents the result of an endpoint check
type EndpointResult struct {
	Check    cfg.Check
	Service  cfg.Service
	RpcUrl   string
	Protocol string
	Domain   string
	Port     string
	Path     string
	Results  []Result
	Mu       sync.RWMutex
}

// Result represents the result profile for a member's check
type Result struct {
	Member    cfg.Member
	Status    bool
	Checktime time.Time
	ErrorText string
	Data      map[string]interface{}
}
