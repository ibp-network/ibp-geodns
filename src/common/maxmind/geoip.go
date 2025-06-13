package maxmind

import (
	"fmt"
	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	"math"
	"net"
	"net/url"
	"os"
	"path/filepath"

	"github.com/oschwald/maxminddb-golang"
)

var (
	maxmindAsn     *maxminddb.Reader
	maxmindCity    *maxminddb.Reader
	maxmindCountry *maxminddb.Reader
)

type URLParts struct {
	Protocol  string
	Domain    string
	Port      string
	Directory string
}

func Init() {
	c := cfg.GetConfig()

	baseDir := filepath.Join(c.Local.Maxmind.MaxmindDBPath)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Log(log.Fatal, "Failed to create maxmind directory %s: %v", baseDir, err)
		os.Exit(1)
	}

	err := updateMaxmindDatabase()
	if err != nil {
		log.Log(log.Error, "Auto-update error: %v", err)
	}

	err = loadLocalDatabases(baseDir)
	if err != nil {
		log.Log(log.Fatal, "Failed to load local maxmind databases: %v", err)
		os.Exit(1)
	}
}

func loadLocalDatabases(baseDir string) error {
	var err error

	cityPath := filepath.Join(baseDir, "CityLite.mmdb")
	countryPath := filepath.Join(baseDir, "CountryLite.mmdb")
	asnPath := filepath.Join(baseDir, "AsnLite.mmdb")

	if _, statErr := os.Stat(cityPath); statErr == nil {
		maxmindCity, err = maxminddb.Open(cityPath)
		if err != nil {
			return fmt.Errorf("could not open city database %s: %w", cityPath, err)
		}
	} else {
		log.Log(log.Error, "CityLite.mmdb not found at %s", cityPath)
	}

	if _, statErr := os.Stat(countryPath); statErr == nil {
		maxmindCountry, err = maxminddb.Open(countryPath)
		if err != nil {
			return fmt.Errorf("could not open country database %s: %w", countryPath, err)
		}
	} else {
		log.Log(log.Error, "CountryLite.mmdb not found at %s", countryPath)
	}

	if _, statErr := os.Stat(asnPath); statErr == nil {
		maxmindAsn, err = maxminddb.Open(asnPath)
		if err != nil {
			return fmt.Errorf("could not open ASN database %s: %w", asnPath, err)
		}
	} else {
		log.Log(log.Error, "AsnLite.mmdb not found at %s", asnPath)
	}

	return nil
}

func Distance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)

	lat1 = lat1 * (math.Pi / 180.0)
	lat2 = lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1)*math.Cos(lat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func GetClientCoordinates(ipStr string) (float64, float64) {
	if maxmindCity == nil {
		log.Log(log.Error, "CityLite is not loaded")
		return 0, 0
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		log.Log(log.Error, "Invalid IP address: %s", ipStr)
		return 0, 0
	}

	var record struct {
		Location struct {
			Latitude  float64 `maxminddb:"latitude"`
			Longitude float64 `maxminddb:"longitude"`
		} `maxminddb:"location"`
	}

	if err := maxmindCity.Lookup(ip, &record); err != nil {
		log.Log(log.Error, "CityLite lookup error: %v", err)
		return 0, 0
	}
	return record.Location.Latitude, record.Location.Longitude
}

func GetCountryCode(ipStr string) string {
	if maxmindCity == nil {
		log.Log(log.Error, "CityLite DB is not loaded, cannot fetch country code.")
		return ""
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		log.Log(log.Error, "Invalid IP address: %s", ipStr)
		return ""
	}

	var record struct {
		Country struct {
			IsoCode string `maxminddb:"iso_code"`
		} `maxminddb:"country"`
	}
	if err := maxmindCity.Lookup(ip, &record); err != nil {
		log.Log(log.Error, "Failed city lookup for IP %s: %v", ipStr, err)
		return ""
	}

	return record.Country.IsoCode
}

func GetCountryName(ipStr string) string {
	if maxmindCity == nil {
		log.Log(log.Error, "CityLite DB not loaded, cannot fetch country name.")
		return ""
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		log.Log(log.Error, "Invalid IP address: %s", ipStr)
		return ""
	}

	var record struct {
		Country struct {
			Names map[string]string `maxminddb:"names"`
		} `maxminddb:"country"`
	}

	if err := maxmindCity.Lookup(ip, &record); err != nil {
		log.Log(log.Error, "Failed city/country lookup for IP %s: %v", ipStr, err)
		return ""
	}

	if name, ok := record.Country.Names["en"]; ok {
		return name
	}
	return ""
}

func GetClassC(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		log.Log(log.Error, "Invalid IP address: %s", ipStr)
		return ""
	}
	ipv4 := ip.To4()
	if ipv4 == nil {
		log.Log(log.Error, "Non-IPv4 address: %s", ipStr)
		return ""
	}
	return fmt.Sprintf("%d.%d.%d", ipv4[0], ipv4[1], ipv4[2])
}

func GetAsnAndNetwork(ipStr string) (string, string) {
	if maxmindAsn == nil {
		return "", ""
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		log.Log(log.Error, "Invalid IP address in GetAsnAndNetwork: %s", ipStr)
		return "", ""
	}

	var record struct {
		AutonomousSystemNumber       uint   `maxminddb:"autonomous_system_number"`
		AutonomousSystemOrganization string `maxminddb:"autonomous_system_organization"`
	}

	if err := maxmindAsn.Lookup(ip, &record); err != nil {
		log.Log(log.Error, "Failed asn lookup for IP %s: %v", ipStr, err)
		return "", ""
	}

	if record.AutonomousSystemNumber == 0 {
		return "", ""
	}

	asn := fmt.Sprintf("AS%d", record.AutonomousSystemNumber)
	return asn, record.AutonomousSystemOrganization
}

func Close() {
	if maxmindCity != nil {
		maxmindCity.Close()
	}
	if maxmindCountry != nil {
		maxmindCountry.Close()
	}
	if maxmindAsn != nil {
		maxmindAsn.Close()
	}
}

func ParseUrl(rawURL string) URLParts {
	u, err := url.Parse(rawURL)
	if err != nil {
		log.Log(log.Error, "Error parsing URL %s", rawURL)
		return URLParts{}
	}

	return URLParts{
		Protocol:  u.Scheme + "://",
		Domain:    u.Hostname(),
		Port:      u.Port(),
		Directory: u.Path,
	}
}
