package geoip

import (
	"common/config"
	l "common/logging"
	"fmt"
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

// Init initializes the GeoIP database reader with the specified database path.
func Init() {
	c := config.GetConfig()
	var err error

	maxmindAsnDB := filepath.Join(c.System.GeoliteDBPath, "Asn.mmdb")
	maxmindAsn, err = maxminddb.Open(maxmindAsnDB)

	if err != nil {
		l.Log(l.Error, "Could not initialize Maxmind ASN Database")
		os.Exit(1)
	}

	maxmindCountryDB := filepath.Join(c.System.GeoliteDBPath, "Country.mmdb")
	maxmindCountry, err = maxminddb.Open(maxmindCountryDB)

	if err != nil {
		l.Log(l.Error, "Could not initialize Maxmind Country Database")
		os.Exit(1)
	}

	maxmindCityDB := filepath.Join(c.System.WorkDir, c.System.GeoliteDBPath, "City.mmdb")
	maxmindCity, err = maxminddb.Open(maxmindCityDB)

	if err != nil {
		l.Log(l.Error, "Could not initialize Maxmind City Database")
		os.Exit(1)
	}
}

// Distance calculates the Haversine distance between two geographic coordinates.
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

// GetClientCoordinates retrieves the latitude and longitude for a given IP address.
func GetClientCoordinates(ipStr string) (float64, float64) {
	var record struct {
		Location struct {
			Latitude  float64 `maxminddb:"latitude"`
			Longitude float64 `maxminddb:"longitude"`
		} `maxminddb:"location"`
	}

	if maxmindCity == nil {
		l.Log(l.Error, "GeoIP database is not initialized")
		return 0, 0
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		l.Log(l.Error, "Invalid IP Address: %s", ipStr)
		return 0, 0
	}

	err := maxmindCity.Lookup(ip, &record)
	if err != nil {
		return 0, 0
	}

	return record.Location.Latitude, record.Location.Longitude
}

// GetCountryCode looks up the country code for the given IP address.
func GetCountryCode(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		l.Log(l.Error, "Invalid IP address: %s", ipStr)
		return ""
	}

	if maxmindCity == nil {
		l.Log(l.Error, "GeoIP reader not initialized")
		return ""
	}

	var record struct {
		Country struct {
			IsoCode string `maxminddb:"iso_code"`
		} `maxminddb:"country"`
	}

	err := maxmindCity.Lookup(ip, &record)
	if err != nil {
		l.Log(l.Error, "Failed to lookup GeoIP data: %v", err)
		return ""
	}

	return record.Country.IsoCode
}

// GetClassC strips the IP address to its Class C subnet (e.g., "192.168.1").
func GetClassC(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		l.Log(l.Error, "Invalid IP address: %s", ipStr)
		return ""
	}

	// Ensure IPv4
	ipv4 := ip.To4()
	if ipv4 == nil {
		l.Log(l.Error, "Non-IPv4 address: %s", ipStr)
		return ""
	}

	classC := fmt.Sprintf("%d.%d.%d", ipv4[0], ipv4[1], ipv4[2])
	return classC
}

// Close cleans up resources used by the stats package.
func Close() {
	if maxmindCity != nil {
		maxmindCity.Close()
	}
}

func ParseUrl(rawURL string) URLParts {
	u, err := url.Parse(rawURL)
	if err != nil {
		l.Log(l.Debug, "Error parsing URL %s", rawURL)
		return URLParts{}
	}

	parts := URLParts{
		Protocol:  u.Scheme + "://",
		Domain:    u.Hostname(),
		Port:      u.Port(),
		Directory: u.Path,
	}

	return parts
}
