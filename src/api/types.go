package api

import (
	"sync"

	cfg "github.com/ibp-network/ibp-geodns-libs/config"
)

type StaticMap struct {
	mu      sync.RWMutex
	records []cfg.DNSRecord
}

var StaticRecords = &StaticMap{
	records: []cfg.DNSRecord{},
}

type TLDMap struct {
	mu      sync.RWMutex
	records map[int]string
}

var TLDRecords = &TLDMap{
	records: make(map[int]string),
}

type ServiceMap struct {
	mu       sync.RWMutex
	Services map[string]ServiceConfigs
}

var ServiceRecords = &ServiceMap{
	Services: make(map[string]ServiceConfigs),
}

type ServiceConfigs struct {
	Name          string
	Active        int
	LevelRequired int
	NetworkName   string
	ServiceType   string
	Members       map[string]cfg.Member
}

type Request struct {
	Method     string     `json:"method"`
	Parameters Parameters `json:"parameters"`
}
type Response struct {
	Result interface{} `json:"result"`
}

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

	MemberName string `json:"memberName"`
	Domain     string `json:"domain"`
	StartTime  string `json:"startTime"`
	EndTime    string `json:"endTime"`
}

type DomainInfo struct {
	DomainID       int      `json:"id"`
	Zone           string   `json:"zone"`
	Masters        []string `json:"masters"`
	NotifiedSerial int      `json:"notified_serial"`
	Serial         int      `json:"serial"`
	LastCheck      int      `json:"last_check"`
	Kind           string   `json:"kind"`
}
