package data

import (
	cfg "ibp-geodns/src/common/config"
	"sync"
	"time"
)

type OfficialResults struct {
	SiteResults     []SiteResult
	DomainResults   []DomainResult
	EndpointResults []EndpointResult
	Mu              sync.RWMutex
}

type LocalResults struct {
	SiteResults     []SiteResult
	DomainResults   []DomainResult
	EndpointResults []EndpointResult
	Mu              sync.RWMutex
}

type Result struct {
	Member    cfg.Member
	Status    bool
	Checktime time.Time
	ErrorText string
	Data      map[string]interface{}
	IsIPv6    bool
}

type SiteResult struct {
	Check   cfg.Check
	IsIPv6  bool
	Results []Result
}

type DomainResult struct {
	Check   cfg.Check
	Service cfg.Service
	Domain  string
	IsIPv6  bool
	Results []Result
}

type EndpointResult struct {
	Check    cfg.Check
	Service  cfg.Service
	RpcUrl   string
	Protocol string
	Domain   string
	Port     string
	Path     string
	IsIPv6   bool
	Results  []Result
}

type StatMap struct {
	Mu   sync.Mutex
	Data map[string]map[string]*DailyStats
}

type DailyStats struct {
	ClientStats ClientStats             `json:"ClientStats"`
	MemberStats map[string]*MemberStats `json:"MemberStats"`
}

type ClientStats struct {
	ClassCs      map[string]int `json:"ClassCs"`
	Countries    map[string]int `json:"Countries"`
	Networks     map[string]int `json:"Networks"`
	CountryNames map[string]int `json:"CountryNames"`
	Asns         map[string]int `json:"Asns"`
	Requests     int            `json:"Requests"`
}

type MemberStats struct {
	ClassCs      map[string]int `json:"ClassCs"`
	Countries    map[string]int `json:"Countries"`
	Networks     map[string]int `json:"Networks"`
	CountryNames map[string]int `json:"CountryNames"`
	Asns         map[string]int `json:"Asns"`
	Requests     int            `json:"Requests"`
}

type Billing struct {
	Records map[string][]EventRecord
	Mu      sync.RWMutex
}

type BCache struct {
	Data     map[string]interface{}
	FilePath string
}

type EventRecord struct {
	CheckType  string                 `json:"CheckType"`
	CheckName  string                 `json:"CheckName"`
	MemberName string                 `json:"MemberName"`
	DomainName string                 `json:"DomainName,omitempty"`
	Endpoint   string                 `json:"Endpoint,omitempty"`
	Status     bool                   `json:"Status"`
	ErrorText  string                 `json:"ErrorText"`
	Data       map[string]interface{} `json:"Data"`
	IsIPv6     bool                   `json:"IsIPv6"`

	StartTime time.Time `json:"StartTime"`
	EndTime   time.Time `json:"EndTime"`

	StartDate string `json:"StartDate"`
	EndDate   string `json:"EndDate"`
}
