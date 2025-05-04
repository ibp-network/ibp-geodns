package config

import (
	"sync"
	"time"
)

// Config holds the configuration data and internal fields
type Config struct {
	mu      sync.RWMutex
	cfgFile string
	data    ConfigData
}

// ConfigData holds the actual configuration data without the mutex
type ConfigData struct {
	System          SystemConfig           `json:"System"`
	StaticDNS       []DNSRecord            `json:"StaticDNS"`
	Members         map[string]Member      `json:"Members"`
	Services        map[string]Service     `json:"Services"`
	Pricing         map[string]IaasPricing `json:"IaasPricing"`
	ServiceRequests ServiceRequests        `json:"ServiceRequests"`
}

// SystemConfig represents the configuration loaded from disk (config.json)
type SystemConfig struct {
	ConfigPath             string        `json:"-"`
	WorkDir                string        `json:"workDir"`
	GeoliteDBPath          string        `json:"GeoliteDBPath"`
	StaticDNSConfig        string        `json:"StaticDNSConfig"`
	MembersConfig          string        `json:"MembersConfig"`
	ServicesConfig         string        `json:"ServicesConfig"`
	IaasPricingConfig      string        `json:"IaasPricingConfig"`
	ServicesRequestsConfig string        `json:"ServicesRequestsConfig"`
	MinimumOfflineTime     int           `json:"MinimumOfflineTime"`
	Nats                   NatsConfig    `json:"Nats"`
	Mysql                  MysqlConfig   `json:"Mysql"`
	DnsApi                 ApiConfig     `json:"DnsApi"`
	MgmtApi                ApiConfig     `json:"MgmtApi"`
	ConfigReloadTime       time.Duration `json:"ConfigReloadTime"`
	CacheSaveTime          time.Duration `json:"CacheSaveTime"`
	Matrix                 MatrixConfig  `json:"Matrix"`
	Checks                 []Check       `json:"Checks"`
}

// IaasPricing represents the pricing details for a region
type IaasPricing struct {
	Cores     float64 `json:"cores"`
	Memory    float64 `json:"memory"`
	Disk      float64 `json:"disk"`
	Bandwidth float64 `json:"bandwidth"`
}

// APIServerConfig represents the API server configuration
type ApiConfig struct {
	ListenAddress string            `json:"ListenAddress"`
	ListenPort    string            `json:"ListenPort"`
	AuthKeys      map[string]string `json:"AuthKeys"`
}

// MatrixConfig represents the Matrix configuration
type MatrixConfig struct {
	Enabled       int    `json:"Enabled"`
	HomeServerURL string `json:"HomeServerURL"`
	Username      string `json:"Username"`
	Password      string `json:"Password"`
	RoomID        string `json:"RoomID"`
}

// Check represents individual check configurations
type Check struct {
	Name          string                 `json:"Name"`
	Enabled       int                    `json:"Enabled"`
	CheckType     string                 `json:"CheckType"`
	Timeout       int                    `json:"Timeout"`
	CheckInterval int                    `json:"CheckInterval"`
	ExtraOptions  map[string]interface{} `json:"ExtraOptions"`
}

// StaticDNSRecord represents a record in the static DNS configuration
type DNSRecord struct {
	QName    string `json:"qname"`
	QType    string `json:"qtype"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Auth     bool   `json:"auth"`
	DomainID int    `json:"domain_id"`
}

// Member represents a member in the members configuration
type Member struct {
	Details            MemberDetails `json:"Details"`
	Membership         Membership    `json:"Membership"`
	Service            ServiceInfo   `json:"Service"`
	Override           bool
	OverrideTime       time.Time
	ServiceAssignments map[string][]string `json:"ServiceAssignments"`
	Location           Location            `json:"Location"`
}

// MemberDetails represents the details of a member
type MemberDetails struct {
	Name    string `json:"Name"`
	Website string `json:"Website"`
	Logo    string `json:"Logo"`
}

// Membership represents the membership information of a member
type Membership struct {
	Level      int `json:"MemberLevel"`
	Joined     int `json:"Joined"`
	LastRankup int `json:"LastRankup"`
}

// ServiceInfo represents the service information of a member
type ServiceInfo struct {
	Active      int    `json:"Active"`
	ServiceIPv4 string `json:"ServiceIPv4"`
	ServiceIPv6 string `json:"ServiceIPv6"`
	MonitorUrl  string `json:"MonitorUrl"`
}

// Location represents the geographical location of a member
type Location struct {
	Region    string  `json:"Region"`
	Latitude  float64 `json:"Latitude"`
	Longitude float64 `json:"Longitude"`
}

// Service represents a service in the services configuration
type Service struct {
	Configuration ServiceConfiguration       `json:"Configuration"`
	Resources     Resources                  `json:"Resources"`
	Providers     map[string]ServiceProvider `json:"Providers"`
}

// ServiceProvider represents a service provider's information
type ServiceProvider struct {
	RpcUrls []string `json:"RpcUrls"`
}

// ServiceConfiguration represents the configuration of a service
type ServiceConfiguration struct {
	Name          string `json:"Name"`
	ServiceType   string `json:"ServiceType"`
	Active        int    `json:"Active"`
	LevelRequired int    `json:"LevelRequired"`
	NetworkName   string `json:"NetworkName"`
}

type Resources struct {
	Nodes     int     `json:"nodes"`
	Cores     float64 `json:"cores"`
	Memory    float64 `json:"memory"`
	Disk      float64 `json:"disk"`
	Bandwidth float64 `json:"bandwidth"`
}

// Define the structure of "dns" and "wss" data
type RequestStats struct {
	Requests        int `json:"requests"`
	UniqueIPs       int `json:"uniqueIPs"`
	UniqueCClass    int `json:"uniqueCClass"`
	UniqueCountries int `json:"uniqueCountries"`
}

// Define the structure of each month, containing "dns" and "wss"
type MonthlyData struct {
	DNS RequestStats `json:"dns"`
	WSS RequestStats `json:"wss"`
}

// Define the overall structure, mapping service names to monthly data
type ServiceRequests struct {
	Requests map[string]map[string]MonthlyData
}

// NodeInfo holds information about a cluster node.
type NatsConfig struct {
	NodeID string `json:"NodeID"`
	User   string `json:"User"`
	Pass   string `json:"Pass"`
	Url    string `json:"Url"`
}

// NodeInfo holds information about a cluster node.
type MysqlConfig struct {
	Host string `json:"Host"`
	Port string `json:"Port"`
	User string `json:"User"`
	Pass string `json:"Pass"`
	DB   string `json:"DB"`
}
