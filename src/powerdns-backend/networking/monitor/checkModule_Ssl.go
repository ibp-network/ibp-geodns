package monitor

import (
	"common/config"
	"crypto/tls"
	"net"
	"time"
)

func init() {
	RegisterDomainCheck("ssl", SslCheck)
}

func SslCheck(check config.Check, domain string, service config.Service, member config.Member) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(member.Service.ServiceIPv4, "443"), time.Duration(getIntOption(check.ExtraOptions, "ConnectTimeout", 5))*time.Second)
	if err != nil {
		go UpdateDomainResultLocal(check, domain, service, member, false, "TCP connection error", nil)
		return
	}
	defer conn.Close()

	tlsConn := tls.Client(conn, &tls.Config{
		ServerName:         domain,
		InsecureSkipVerify: false,
	})
	err = tlsConn.Handshake()
	if err != nil {
		go UpdateDomainResultLocal(check, domain, service, member, false, "TLS handshake failed", nil)
		return
	}
	defer tlsConn.Close()

	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		go UpdateDomainResultLocal(check, domain, service, member, false, "No certificates found", nil)
		return
	}

	cert := certs[0]
	daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / 24)

	var success bool
	var errortext string

	if daysUntilExpiry < 5 {
		success = false
		errortext = "Less than 5 days until expiry"
	} else {
		success = true
		errortext = ""
	}

	// Prepare data for the result
	data := map[string]interface{}{
		"ExpiryTimestamp": cert.NotAfter.Unix(),
		"DaysUntilExpiry": daysUntilExpiry,
	}

	go UpdateDomainResultLocal(check, domain, service, member, success, errortext, data)
}
