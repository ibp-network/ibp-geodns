package config

type CheckConfig struct {
	Enabled       int                    `json:"Enabled"`
	CheckType     string                 `json:"CheckType"`
	Timeout       int                    `json:"Timeout"`
	CheckInterval int                    `json:"CheckInterval"`
	ExtraOptions  map[string]interface{} `json:"ExtraOptions"`
}

type Config struct {
	ServerName         string                 `json:"ServerName"`
	GeoliteDBPath      string                 `json:"GeoliteDBPath"`
	StaticDNSConfigUrl string                 `json:"StaticDNSConfigUrl"`
	MembersConfigUrl   string                 `json:"MembersConfigUrl"`
	ServicesConfigUrl  string                 `json:"ServicesConfigUrl"`
	MinimumOfflineTime int                    `json:"MinimumOfflineTime"`
	AuthKey            map[string]string      `json:"AuthKey"`
	Matrix             *Matrix                `json:"Matrix"`
	Checks             map[string]CheckConfig `json:"Checks"`
	Prometheus         *PrometheusConfig      `json:"Prometheus"`
}

type Matrix struct {
	Enabled       int    `json:"Enabled"`
	HomeServerURL string `json:"HomeServerURL"`
	Username      string `json:"Username"`
	Password      string `json:"Password"`
	RoomID        string `json:"RoomID"`
}

type SiteCheckResult struct {
	CheckName  string                 `json:"checkname"`
	Success    bool                   `json:"success"`
	CheckError string                 `json:"checkerror,omitempty"`
	CheckData  map[string]interface{} `json:"checkdata,omitempty"`
}

type EndpointCheckResult struct {
	CheckName  string                 `json:"checkname"`
	Success    bool                   `json:"success"`
	CheckError string                 `json:"checkerror,omitempty"`
	CheckData  map[string]interface{} `json:"checkdata,omitempty"`
}

type SiteResults struct {
	ResultType string                                `json:"resulttype"`
	Members    map[string]map[string]SiteCheckResult `json:"members"`
}

type EndpointResults struct {
	ResultType string                                               `json:"resulttype"`
	Endpoint   map[string]map[string]map[string]EndpointCheckResult `json:"endpoint"`
}

type Member struct {
	Details struct {
		Name    string `json:"Name"`
		Website string `json:"Website"`
		Logo    string `json:"Logo"`
	} `json:"Details"`
	Membership struct {
		MemberLevel int `json:"MemberLevel"`
		Joined      int `json:"Joined"`
		LastRankup  int `json:"LastRankup"`
	} `json:"Membership"`
	Service struct {
		Active      int    `json:"Active"`
		ServiceIPv4 string `json:"ServiceIPv4"`
		ServiceIPv6 string `json:"ServiceIPv6"`
		MonitorUrl  string `json:"MonitorUrl"`
	} `json:"Service"`
	ServiceAssignments map[string][]string `json:"ServiceAssignments"`
	Location           struct {
		Region    string  `json:"Region"`
		Latitude  float64 `json:"Latitude"`
		Longitude float64 `json:"Longitude"`
	} `json:"Location"`
}

type Service struct {
	Configuration struct {
		ServiceType   string `json:"ServiceType"`
		Active        int    `json:"Active"`
		LevelRequired int    `json:"LevelRequired"`
		NetworkName   string `json:"NetworkName"`
	} `json:"Configuration"`
	Providers map[string]struct {
		RpcUrls []string `json:"RpcUrls"`
	} `json:"Providers"`
}

type Endpoint struct {
	MemberName      string
	IPv4            string
	IPv6            string
	ExpectedNetwork string
	Latitude        float64
	Longitude       float64
	OriginalURLs    []OriginalURL
}

type OriginalURL struct {
	URL         string
	NetworkName string
}

type MemberService struct {
	IPv4Addresses []string
	IPv6Addresses []string
	Services      []string
}

type ServiceEndpoint struct {
	ExpectedNetwork string
	URLs            []OriginalURL
	ServiceIPv4s    []string
	ServiceIPv6s    []string
	Domains         []string
}

type PrometheusConfig struct {
	Enabled bool `json:"Enabled"`
	UpdateIntervalSeconds int `json:"UpdateIntervalSeconds"`
}
