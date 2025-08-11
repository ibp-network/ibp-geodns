package config

import (
	"sync"
	"time"
)

type ConfigInit struct {
	mu      sync.RWMutex
	cfgFile string
	data    Config
}

type Config struct {
	Local           LocalConfig            `json:"System"`
	StaticDNS       []DNSRecord            `json:"StaticDNS"`
	Members         map[string]Member      `json:"Members"`
	Services        map[string]Service     `json:"Services"`
	Pricing         map[string]IaasPricing `json:"IaasPricing"`
	ServiceRequests ServiceRequests        `json:"ServiceRequests"`
	Alerts          AlertsConfig           `json:"Alerts"`
}

type LocalConfig struct {
	System       SystemConfig  `json:"System"`
	Maxmind      MaxmindConfig `json:"Maxmind"`
	Nats         NatsConfig    `json:"Nats"`
	Mysql        MysqlConfig   `json:"Mysql"`
	DnsApi       ApiConfig     `json:"DnsApi"`
	CollatorApi  ApiConfig     `json:"CollatorApi"`
	MonitorApi   ApiConfig     `json:"MonitorApi"`
	MgmtApi      ApiConfig     `json:"MgmtApi"`
	Discord      DiscordConfig
	Matrix       MatrixConfig
	CheckWorkers CheckWorkers `json:"CheckWorkers"`
	Checks       []Check      `json:"Checks"`
}

type CheckWorkers struct {
	NumWorkers         int `json:"numWorkers"`
	SeparationInterval int `json:"separationInterval"`
}

type DiscordConfig struct {
	Token string `json:"Token"`
}

type SystemConfig struct {
	WorkDir            string        `json:"WorkDir"`
	LogLevel           string        `json:"LogLevel"`
	ConfigReloadTime   time.Duration `json:"ConfigReloadTime"`
	CacheSaveTime      time.Duration `json:"CacheSaveTime"`
	MinimumOfflineTime int           `json:"MinimumOfflineTime"`
	ConfigUrls         ConfigUrls    `json:"ConfigUrls"`
}

type ConfigUrls struct {
	StaticDNSConfig        string `json:"StaticDNSConfig"`
	MembersConfig          string `json:"MembersConfig"`
	ServicesConfig         string `json:"ServicesConfig"`
	IaasPricingConfig      string `json:"IaasPricingConfig"`
	ServicesRequestsConfig string `json:"ServicesRequestsConfig"`
	AlertsConfig           string `json:"AlertsConfig"`
}

type AlertsConfig struct {
	Matrix struct {
		Room         string              `json:"room"`
		InternalRoom string              `json:"internal_room"`
		Members      map[string][]string `json:"members"`
	} `json:"matrix"`
}

type IaasPricing struct {
	Cores     float64 `json:"cores"`
	Memory    float64 `json:"memory"`
	Disk      float64 `json:"disk"`
	Bandwidth float64 `json:"bandwidth"`
}

type ApiConfig struct {
	ListenAddress          string            `json:"ListenAddress"`
	ListenPort             string            `json:"ListenPort"`
	MonitorAddress         string            `json:"MonitorAddress"`
	MonitorPort            string            `json:"MonitorPort"`
	AuthKeys               map[string]string `json:"AuthKeys"`
	RefreshIntervalSeconds int               `json:"RefreshIntervalSeconds"`
}

type MatrixConfig struct {
	HomeServerURL string `json:"HomeServerURL"`
	Username      string `json:"Username"`
	Password      string `json:"Password"`
	RoomID        string `json:"RoomID"`
}

type Check struct {
	Name            string                 `json:"Name"`
	Enabled         int                    `json:"Enabled"`
	CheckType       string                 `json:"CheckType"`
	Timeout         int                    `json:"Timeout"`
	MinimumInterval int                    `json:"minimumInterval"`
	ExtraOptions    map[string]interface{} `json:"ExtraOptions"`
}

type DNSRecord struct {
	QName    string `json:"qname"`
	QType    string `json:"qtype"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Auth     bool   `json:"auth"`
	DomainID int    `json:"domain_id"`
}

type Member struct {
	Details            MemberDetails `json:"Details"`
	Membership         Membership    `json:"Membership"`
	Service            ServiceInfo   `json:"Service"`
	Override           bool
	OverrideTime       time.Time
	ServiceAssignments map[string][]string `json:"ServiceAssignments"`
	Location           Location            `json:"Location"`
}

type MemberDetails struct {
	Name    string `json:"Name"`
	Website string `json:"Website"`
	Logo    string `json:"Logo"`
}

type Membership struct {
	Level      int `json:"MemberLevel"`
	Joined     int `json:"Joined"`
	LastRankup int `json:"LastRankup"`
}

type ServiceInfo struct {
	Active      int    `json:"Active"`
	ServiceIPv4 string `json:"ServiceIPv4"`
	ServiceIPv6 string `json:"ServiceIPv6"`
	MonitorUrl  string `json:"MonitorUrl"`
}

type Location struct {
	Region    string  `json:"Region"`
	Latitude  float64 `json:"Latitude"`
	Longitude float64 `json:"Longitude"`
}

type Service struct {
	Configuration ServiceConfiguration       `json:"Configuration"`
	Resources     Resources                  `json:"Resources"`
	Providers     map[string]ServiceProvider `json:"Providers"`
}

type ServiceProvider struct {
	RpcUrls []string `json:"RpcUrls"`
}

type ServiceConfiguration struct {
	Name          string `json:"Name"`
	ServiceType   string `json:"ServiceType"`
	Active        int    `json:"Active"`
	LevelRequired int    `json:"LevelRequired"`
	NetworkName   string `json:"NetworkName"`
	RelayNetwork  string `json:"RelayNetwork"`
	NetworkType   string `json:"NetworkType"`
	DisplayName   string `json:"DisplayName"`
	WebsiteURL    string `json:"WebsiteURL"`
	LogoURL       string `json:"LogoURL"`
	Description   string `json:"Description"`
	StateRootHash string `json:"StateRootHash"`
}
type Resources struct {
	Nodes     int     `json:"nodes"`
	Cores     float64 `json:"cores"`
	Memory    float64 `json:"memory"`
	Disk      float64 `json:"disk"`
	Bandwidth float64 `json:"bandwidth"`
}

type ServiceRequests struct {
	Requests map[string]map[string]MonthlyData
}

type MonthlyData struct {
	DNS RequestStats `json:"dns"`
	WSS RequestStats `json:"wss"`
}

type RequestStats struct {
	Requests        int `json:"requests"`
	UniqueIPs       int `json:"uniqueIPs"`
	UniqueCClass    int `json:"uniqueCClass"`
	UniqueCountries int `json:"uniqueCountries"`
}

type NatsConfig struct {
	NodeID string `json:"NodeID"`
	User   string `json:"User"`
	Pass   string `json:"Pass"`
	Url    string `json:"Url"`
}

type MaxmindConfig struct {
	MaxmindDBPath string `json:"MaxmindDBPath"`
	AccountID     string `json:"AccountID"`
	LicenseKey    string `json:"LicenseKey"`
}

type MysqlConfig struct {
	Host string `json:"Host"`
	Port string `json:"Port"`
	User string `json:"User"`
	Pass string `json:"Pass"`
	DB   string `json:"DB"`
}
