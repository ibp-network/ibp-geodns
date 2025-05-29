package monitor

import (
	"crypto/tls"
	"net"
	"time"

	cfg "ibp-geodns/src/common/config"
)

// We register the SSL check under name "ssl" so it is invoked if
// the config has a domain-level check with { "Name": "ssl", "CheckType": "domain" }.
func init() {
	RegisterDomainCheck("ssl", SslCheck)
}

// SslCheck tries IPv4 first, then IPv6 (if present). It attempts a
// TLS handshake on port 443, inspects the certificate for upcoming expiry.
func SslCheck(check cfg.Check, domain string, service cfg.Service, member cfg.Member) {
	ip4 := member.Service.ServiceIPv4
	ip6 := member.Service.ServiceIPv6

	// If no IP is configured, fail immediately.
	if ip4 == "" && ip6 == "" {
		UpdateDomainResultLocal(check, domain, service, member, false, "No IPv4 or IPv6 configured", nil)
		return
	}

	success := false
	errText := ""
	dataMap := map[string]interface{}{}

	// Attempt IPv4 if present.
	if ip4 != "" {
		dialErr := dialAndCheckTLS(check, domain, service, member, ip4, &success, &errText, dataMap)
		// If success is true (and no dialErr) => we are done.
		if dialErr == nil && success {
			return
		}
		// Otherwise, we try IPv6 if it exists.
	}

	// If still not success and ip6 is available, try IPv6
	if !success && ip6 != "" {
		dialErr := dialAndCheckTLS(check, domain, service, member, ip6, &success, &errText, dataMap)
		if dialErr == nil && success {
			return
		}
	}

	// If we got here => not successful
	UpdateDomainResultLocal(check, domain, service, member, false, errText, dataMap)
}

// dialAndCheckTLS tries a connection to ip:443, verifies the TLS handshake,
// sets success=false if the cert is nearly expired, or if handshake fails.
func dialAndCheckTLS(
	check cfg.Check,
	domain string,
	service cfg.Service,
	member cfg.Member,
	ip string,
	success *bool,
	errText *string,
	dataMap map[string]interface{},
) error {

	conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, "443"),
		time.Duration(getIntOption(check.ExtraOptions, "ConnectTimeout", 5))*time.Second)
	if err != nil {
		*errText = "TCP connect error: " + err.Error()
		return err
	}
	defer conn.Close()

	tlsConn := tls.Client(conn, &tls.Config{
		ServerName:         domain,
		InsecureSkipVerify: false,
	})
	err = tlsConn.Handshake()
	if err != nil {
		*errText = "TLS handshake failed: " + err.Error()
		return err
	}
	defer tlsConn.Close()

	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		*errText = "No cert found"
		return nil // No error, but success stays false by default
	}

	cert := certs[0]
	daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / 24)

	// Mark success = true for now
	*success = true
	*errText = ""

	// If the certificate expires in < 5 days, we consider that "unhealthy".
	if daysUntilExpiry < 5 {
		*success = false
		*errText = "Less than 5 days to expiry"
	}

	dataMap["ExpiryTimestamp"] = cert.NotAfter.Unix()
	dataMap["DaysUntilExpiry"] = daysUntilExpiry

	// If still success => we call UpdateDomainResultLocal now for an immediate "true".
	if *success {
		UpdateDomainResultLocal(check, domain, service, member, true, "", dataMap)
	}
	return nil
}
