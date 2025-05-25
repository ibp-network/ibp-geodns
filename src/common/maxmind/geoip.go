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

// Global handles to mmdb Readers, loaded after updating from the website
var (
	maxmindAsn     *maxminddb.Reader
	maxmindCity    *maxminddb.Reader
	maxmindCountry *maxminddb.Reader
)

// URLParts is just a simple breakdown of a parsed URL
type URLParts struct {
	Protocol  string
	Domain    string
	Port      string
	Directory string
}

// Init is called once on startup. It triggers the auto-update procedure,
// then opens each .mmdb file (CityLite, CountryLite, AsnLite) into memory.
func Init() {
	c := cfg.GetConfig()

	// Step 1: Make sure "workDir/tmp/maxmind/" exists
	baseDir := filepath.Join(c.Local.Maxmind.MaxmindDBPath)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Log(log.Fatal, "Failed to create maxmind directory %s: %v", baseDir, err)
		os.Exit(1)
	}

	// Step 2: Download/update each mmdb if needed
	err := updateMaxmindDatabase()
	if err != nil {
		log.Log(log.Error, "Auto-update error: %v", err)
		// Not fatal: we can still attempt to load if older mmdb is present
	}

	// Step 3: Attempt to open each local .mmdb
	err = loadLocalDatabases(baseDir)
	if err != nil {
		log.Log(log.Fatal, "Failed to load local maxmind databases: %v", err)
		os.Exit(1)
	}
}

// loadLocalDatabases attempts to open the (already-downloaded) .mmdb files
// CityLite.mmdb, CountryLite.mmdb, AsnLite.mmdb
func loadLocalDatabases(baseDir string) error {
	var err error

	cityPath := filepath.Join(baseDir, "CityLite.mmdb")
	countryPath := filepath.Join(baseDir, "CountryLite.mmdb")
	asnPath := filepath.Join(baseDir, "AsnLite.mmdb")

	// City
	if _, statErr := os.Stat(cityPath); statErr == nil {
		maxmindCity, err = maxminddb.Open(cityPath)
		if err != nil {
			return fmt.Errorf("could not open city database %s: %w", cityPath, err)
		}
	} else {
		log.Log(log.Error, "CityLite.mmdb not found at %s", cityPath)
	}

	// Country
	if _, statErr := os.Stat(countryPath); statErr == nil {
		maxmindCountry, err = maxminddb.Open(countryPath)
		if err != nil {
			return fmt.Errorf("could not open country database %s: %w", countryPath, err)
		}
	} else {
		log.Log(log.Error, "CountryLite.mmdb not found at %s", countryPath)
	}

	// ASN
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

// Distance calculates the Haversine distance between two geographic coordinates in kilometers.
func Distance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth radius in kilometers
	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)

	lat1 = lat1 * (math.Pi / 180.0)
	lat2 = lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1)*math.Cos(lat2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// GetClientCoordinates retrieves lat/long from the CityLite database
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

// GetCountryCode retrieves the ISO country code from the CityLite database
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

// GetCountryName retrieves the full country name (if available) from the CityLite database
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

	// The "names" map can have multiple localizations; try "en" or fallback
	if name, ok := record.Country.Names["en"]; ok {
		return name
	}
	return ""
}

// GetClassC strips an IPv4 address to the first 3 octets: e.g. 192.168.1
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

// GetAsnAndNetwork retrieves the ASN number and the associated organization name from the AsnLite database.
func GetAsnAndNetwork(ipStr string) (string, string) {
	if maxmindAsn == nil {
		// Possibly return empty if the AsnLite DB not loaded
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
		// Means not found, or private IP
		return "", ""
	}

	asn := fmt.Sprintf("AS%d", record.AutonomousSystemNumber)
	return asn, record.AutonomousSystemOrganization
}

// Close frees resources used by maxmind. (If needed)
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

// ParseUrl is used elsewhere in your code, unchanged
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
