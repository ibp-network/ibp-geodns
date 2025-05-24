package monitor

import (
	"crypto/tls"
	"net"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
)

func init() {
	RegisterDomainCheck("ssl", SslCheck)
}

func SslCheck(check cfg.Check, domain string, service cfg.Service, member cfg.Member) {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(member.Service.ServiceIPv4, "443"),
		time.Duration(getIntOption(check.ExtraOptions, "ConnectTimeout", 5))*time.Second)
	if err != nil {
		UpdateDomainResultLocal(check, domain, service, member, false, "TCP connect error", nil)
		return
	}
	defer conn.Close()

	tlsConn := tls.Client(conn, &tls.Config{
		ServerName:         domain,
		InsecureSkipVerify: false,
	})
	err = tlsConn.Handshake()
	if err != nil {
		UpdateDomainResultLocal(check, domain, service, member, false, "TLS handshake failed", nil)
		return
	}
	defer tlsConn.Close()

	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		UpdateDomainResultLocal(check, domain, service, member, false, "No cert found", nil)
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

	UpdateDomainResultLocal(check, domain, service, member, success, errText, dataMap)
	log.Log(log.Debug, "SSL check domain=%s success=%v", domain, success)
}
