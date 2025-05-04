package data

import (
	"ibp-geodns/config"
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
	Member    config.Member
	Status    bool
	Checktime time.Time
	ErrorText string
	Data      map[string]interface{}
}

type SiteResult struct {
	Check   config.Check
	Results []Result
}

type DomainResult struct {
	Check   config.Check
	Service config.Service
	Domain  string
	Results []Result
}

type EndpointResult struct {
	Check    config.Check
	Service  config.Service
	RpcUrl   string
	Protocol string
	Domain   string
	Port     string
	Path     string
	Results  []Result
}

// StatMap holds the mutex and the data map for statistics collection.
type StatMap struct {
	Mu   sync.Mutex
	Data map[string]map[string]*DailyStats
}

type DailyStats struct {
	ClientStats ClientStats             `json:"ClientStats"`
	MemberStats map[string]*MemberStats `json:"MemberStats"`
}

type ClientStats struct {
	ClassCs   map[string]int `json:"ClassCs"`
	Countries map[string]int `json:"Countries"`
	Requests  int            `json:"Requests"`
}

type MemberStats struct {
	ClassCs   map[string]int `json:"ClassCs"`
	Countries map[string]int `json:"Countries"`
	Requests  int            `json:"Requests"`
}

type Billing struct {
	Records map[string][]EventRecord
	Mu      sync.RWMutex
}

type BCache struct {
	Data     map[string]interface{}
	FilePath string
}

// EventRecord represents a downtime/uptime event for a member.
type EventRecord struct {
	CheckType  string                 `json:"CheckType"`
	CheckName  string                 `json:"CheckName"`
	MemberName string                 `json:"MemberName"`
	DomainName string                 `json:"DomainName,omitempty"`
	Endpoint   string                 `json:"Endpoint,omitempty"`
	Status     bool                   `json:"Status"`
	ErrorText  string                 `json:"ErrorText"`
	Data       map[string]interface{} `json:"Data"`

	StartTime time.Time `json:"StartTime"`
	EndTime   time.Time `json:"EndTime"`

	StartDate string `json:"StartDate"`
	EndDate   string `json:"EndDate"`
}
