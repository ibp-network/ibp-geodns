package config

import (
	"sync"
	"time"
)

// ConfigInit holds the main config pointer with a mutex
type ConfigInit struct {
	mu      sync.RWMutex
	cfgFile string
	data    Config
}

// Config holds all top-level configuration sections
type Config struct {
	Local           LocalConfig            `json:"System"`
	StaticDNS       []DNSRecord            `json:"StaticDNS"`
	Members         map[string]Member      `json:"Members"`
	Services        map[string]Service     `json:"Services"`
	Pricing         map[string]IaasPricing `json:"IaasPricing"`
	ServiceRequests ServiceRequests        `json:"ServiceRequests"`
}

// LocalConfig is loaded from disk (config.json)
type LocalConfig struct {
	System  SystemConfig  `json:"System"`
	Maxmind MaxmindConfig `json:"Maxmind"`
	Nats    NatsConfig    `json:"Nats"`
	Mysql   MysqlConfig   `json:"Mysql"`
	DnsApi  ApiConfig     `json:"DnsApi"`
	// Added new MonitorApi field
	MonitorApi ApiConfig `json:"MonitorApi"`
	MgmtApi    ApiConfig `json:"MgmtApi"`
	Discord    DiscordConfig
	Matrix     MatrixConfig
	Checks     []Check `json:"Checks"`
}

// DiscordConfig holds Discord bot credentials
type DiscordConfig struct {
	Token string `json:"Token"`
}

// SystemConfig for base paths and intervals
type SystemConfig struct {
	WorkDir            string        `json:"workDir"`
	ConfigReloadTime   time.Duration `json:"ConfigReloadTime"`
	CacheSaveTime      time.Duration `json:"CacheSaveTime"`
	MinimumOfflineTime int           `json:"MinimumOfflineTime"`
	ConfigUrls         ConfigUrls    `json:"ConfigUrls"`
}

// ConfigUrls for external JSON fetch
type ConfigUrls struct {
	StaticDNSConfig        string `json:"StaticDNSConfig"`
	MembersConfig          string `json:"MembersConfig"`
	ServicesConfig         string `json:"ServicesConfig"`
	IaasPricingConfig      string `json:"IaasPricingConfig"`
	ServicesRequestsConfig string `json:"ServicesRequestsConfig"`
}

// IaasPricing holds region-based pricing
type IaasPricing struct {
	Cores     float64 `json:"cores"`
	Memory    float64 `json:"memory"`
	Disk      float64 `json:"disk"`
	Bandwidth float64 `json:"bandwidth"`
}

// ApiConfig for HTTP servers
type ApiConfig struct {
	ListenAddress          string            `json:"ListenAddress"`
	ListenPort             string            `json:"ListenPort"`
	AuthKeys               map[string]string `json:"AuthKeys"`
	RefreshIntervalSeconds int               `json:"RefreshIntervalSeconds"` // new
}

// MatrixConfig for matrix bot
type MatrixConfig struct {
	HomeServerURL string `json:"HomeServerURL"`
	Username      string `json:"Username"`
	Password      string `json:"Password"`
	RoomID        string `json:"RoomID"`
}

// Check defines a monitor check
type Check struct {
	Name          string                 `json:"Name"`
	Enabled       int                    `json:"Enabled"`
	CheckType     string                 `json:"CheckType"`
	Timeout       int                    `json:"Timeout"`
	CheckInterval int                    `json:"CheckInterval"`
	ExtraOptions  map[string]interface{} `json:"ExtraOptions"`
}

// DNSRecord for static DNS config
type DNSRecord struct {
	QName    string `json:"qname"`
	QType    string `json:"qtype"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Auth     bool   `json:"auth"`
	DomainID int    `json:"domain_id"`
}

// Member structure
type Member struct {
	Details            MemberDetails `json:"Details"`
	Membership         Membership    `json:"Membership"`
	Service            ServiceInfo   `json:"Service"`
	Override           bool
	OverrideTime       time.Time
	ServiceAssignments map[string][]string `json:"ServiceAssignments"`
	Location           Location            `json:"Location"`
}

// MemberDetails for display info
type MemberDetails struct {
	Name    string `json:"Name"`
	Website string `json:"Website"`
	Logo    string `json:"Logo"`
}

// Membership level info
type Membership struct {
	Level      int `json:"MemberLevel"`
	Joined     int `json:"Joined"`
	LastRankup int `json:"LastRankup"`
}

// ServiceInfo for a member's main Service
type ServiceInfo struct {
	Active      int    `json:"Active"`
	ServiceIPv4 string `json:"ServiceIPv4"`
	ServiceIPv6 string `json:"ServiceIPv6"`
	MonitorUrl  string `json:"MonitorUrl"`
}

// Location represents lat/long for geo
type Location struct {
	Region    string  `json:"Region"`
	Latitude  float64 `json:"Latitude"`
	Longitude float64 `json:"Longitude"`
}

// Service definition
type Service struct {
	Configuration ServiceConfiguration       `json:"Configuration"`
	Resources     Resources                  `json:"Resources"`
	Providers     map[string]ServiceProvider `json:"Providers"`
}

// ServiceProvider data
type ServiceProvider struct {
	RpcUrls []string `json:"RpcUrls"`
}

// ServiceConfiguration data
type ServiceConfiguration struct {
	Name          string `json:"Name"`
	ServiceType   string `json:"ServiceType"`
	Active        int    `json:"Active"`
	LevelRequired int    `json:"LevelRequired"`
	NetworkName   string `json:"NetworkName"`
}

// Resources define service usage
type Resources struct {
	Nodes     int     `json:"nodes"`
	Cores     float64 `json:"cores"`
	Memory    float64 `json:"memory"`
	Disk      float64 `json:"disk"`
	Bandwidth float64 `json:"bandwidth"`
}

// ServiceRequests for monthly usage
type ServiceRequests struct {
	Requests map[string]map[string]MonthlyData
}

// MonthlyData for DNS / WSS
type MonthlyData struct {
	DNS RequestStats `json:"dns"`
	WSS RequestStats `json:"wss"`
}

// RequestStats track usage
type RequestStats struct {
	Requests        int `json:"requests"`
	UniqueIPs       int `json:"uniqueIPs"`
	UniqueCClass    int `json:"uniqueCClass"`
	UniqueCountries int `json:"uniqueCountries"`
}

// SignalConfig for NATS
type NatsConfig struct {
	NodeID string `json:"NodeID"`
	User   string `json:"User"`
	Pass   string `json:"Pass"`
	Url    string `json:"Url"`
}

// MaxmindConfig keys
type MaxmindConfig struct {
	MaxmindDBPath string `json:"MaxmindDBPath"`
	AccountID     string `json:"AccountID"`
	LicenseKey    string `json:"LicenseKey"`
}

// MysqlConfig for DB
type MysqlConfig struct {
	Host string `json:"Host"`
	Port string `json:"Port"`
	User string `json:"User"`
	Pass string `json:"Pass"`
	DB   string `json:"DB"`
}
