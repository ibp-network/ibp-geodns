package monitor

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
)

func init() {
	// SSL check is valid for both RPC and ETHRPC service types
	RegisterDomainCheckWithTypes("ssl", SslCheck, []string{"RPC", "ETHRPC"})
}

func SslCheck(check cfg.Check, domain string, service cfg.Service, member cfg.Member) {
	ip4 := member.Service.ServiceIPv4
	ip6 := member.Service.ServiceIPv6
	if ip4 != "" {
		dialAndCheckTLS(check, domain, service, member, ip4, false)
	}
	if ip6 != "" {
		dialAndCheckTLS(check, domain, service, member, ip6, true)
	}
}

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
		UpdateDomainResultLocal(check, domain, service, member, true, "", dataMap, isIPv6)
		log.Log(log.Debug, "SSL check completed for %s %s isIPv6=%v success=%v", member.Details.Name, domain, isIPv6, true)
	} else {
		UpdateDomainResultLocal(check, domain, service, member, false, errText, dataMap, isIPv6)
		log.Log(log.Debug, "SSL check failed for %s %s isIPv6=%v success=%v", member.Details.Name, domain, isIPv6, false)
	}
}
