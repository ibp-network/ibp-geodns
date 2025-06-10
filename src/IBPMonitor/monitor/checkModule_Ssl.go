package monitor

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	cfg "ibp-geodns/src/common/config"
)

// We register the SSL check under name "ssl" so it is invoked if
// the config has a domain-level check with { "Name": "ssl", "CheckType": "domain" }.
func init() {
	RegisterDomainCheck("ssl", SslCheck)
}

// SslCheck tries IPv4 if present, then IPv6 if present, similar to the ping module.
func SslCheck(check cfg.Check, domain string, service cfg.Service, member cfg.Member) {
	ip4 := member.Service.ServiceIPv4
	ip6 := member.Service.ServiceIPv6

	// If IPv4 is present, do an SSL check on IPv4
	if ip4 != "" {
		dialAndCheckTLS(check, domain, service, member, ip4, false)
	}

	// If IPv6 is present, do an SSL check on IPv6
	if ip6 != "" {
		dialAndCheckTLS(check, domain, service, member, ip6, true)
	}
}

// dialAndCheckTLS tries a connection to ip:443, verifies the TLS handshake, etc.
func dialAndCheckTLS(
	check cfg.Check,
	domain string,
	service cfg.Service,
	member cfg.Member,
	ip string,
	isIPv6 bool,
) {
	timeoutSec := getIntOption(check.ExtraOptions, "ConnectTimeout", 5)
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, "443"), time.Duration(timeoutSec)*time.Second)
	if err != nil {
		UpdateDomainResultLocal(check, domain, service, member, false,
			fmt.Sprintf("TCP connect error: %v", err), nil, isIPv6)
		return
	}
	defer conn.Close()

	tlsConn := tls.Client(conn, &tls.Config{
		ServerName:         domain,
		InsecureSkipVerify: false,
	})
	err = tlsConn.Handshake()
	if err != nil {
		UpdateDomainResultLocal(check, domain, service, member, false,
			fmt.Sprintf("TLS handshake failed: %v", err), nil, isIPv6)
		return
	}
	defer tlsConn.Close()

	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		UpdateDomainResultLocal(check, domain, service, member, false, "No certificate found", nil, isIPv6)
		return
	}

	cert := certs[0]
	daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / 24)

	success := true
	errText := ""
	if daysUntilExpiry < 5 {
		success = false
		errText = "Less than 5 days to expiry"
	}

	dataMap := map[string]interface{}{
		"ExpiryTimestamp": cert.NotAfter.Unix(),
		"DaysUntilExpiry": daysUntilExpiry,
	}
	if success {
		// Mark success
		UpdateDomainResultLocal(check, domain, service, member, true, "", dataMap, isIPv6)
	} else {
		UpdateDomainResultLocal(check, domain, service, member, false, errText, dataMap, isIPv6)
	}
}
