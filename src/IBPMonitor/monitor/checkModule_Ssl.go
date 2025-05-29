package monitor

import (
	"crypto/tls"
	"net"
	"time"

	cfg "ibp-geodns/src/common/config"
)

// SslCheck is registered for "ssl" domain checks (see init() below).
func init() {
	RegisterDomainCheck("ssl", SslCheck)
}

// SslCheck runs an SSL check on port 443 for either IPv4 or IPv6 (fallback).
func SslCheck(check cfg.Check, domain string, service cfg.Service, member cfg.Member) {
	ip4 := member.Service.ServiceIPv4
	ip6 := member.Service.ServiceIPv6

	// We'll attempt IPv4 first if available, else fallback to IPv6 if present.
	if ip4 == "" && ip6 == "" {
		UpdateDomainResultLocal(check, domain, service, member, false, "No IPv4 or IPv6 configured", nil)
		return
	}

	success := false
	errText := ""
	dataMap := make(map[string]interface{})

	// We'll try dialIPv4 first if ip4 is not empty
	if ip4 != "" {
		dialErr := dialAndCheckTLS(check, domain, service, member, ip4, &success, &errText, dataMap)
		if dialErr == nil && success {
			return // IPv4 success => done
		}
		// otherwise, we attempt IPv6 if available
	}

	// If we haven't succeeded and ip6 is available, do IPv6
	if !success && ip6 != "" {
		dialErr := dialAndCheckTLS(check, domain, service, member, ip6, &success, &errText, dataMap)
		if dialErr == nil && success {
			return
		}
	}

	// If still not successful
	UpdateDomainResultLocal(check, domain, service, member, false, errText, dataMap)
}

// dialAndCheckTLS attempts to connect to the given ip on port 443, verify TLS, etc.
func dialAndCheckTLS(check cfg.Check, domain string, service cfg.Service, member cfg.Member, ip string,
	success *bool, errText *string, dataMap map[string]interface{}) error {

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
		return nil
	}

	cert := certs[0]
	daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / 24)

	*success = true
	*errText = ""
	if daysUntilExpiry < 5 {
		*success = false
		*errText = "Less than 5 days to expiry"
	}
	dataMap["ExpiryTimestamp"] = cert.NotAfter.Unix()
	dataMap["DaysUntilExpiry"] = daysUntilExpiry

	if *success {
		UpdateDomainResultLocal(check, domain, service, member, true, "", dataMap)
	}
	return nil
}
