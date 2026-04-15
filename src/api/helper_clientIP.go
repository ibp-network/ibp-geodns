package api

import (
	"net"
	"strings"
)

func getEffectiveClientIP(params Parameters) (net.IP, string) {
	for _, candidate := range []string{params.RealRemote, params.Remote} {
		ip := parseClientIP(candidate)
		if ip != nil {
			return ip, ip.String()
		}
	}
	return nil, ""
}

func parseClientIP(raw string) net.IP {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return nil
	}

	if ip := parseIPLiteral(candidate); ip != nil {
		return ip
	}

	if host, _, err := net.SplitHostPort(candidate); err == nil {
		return parseIPLiteral(host)
	}

	if strings.Count(candidate, ":") == 1 && !strings.Contains(candidate, "]") {
		host, _, ok := strings.Cut(candidate, ":")
		if ok {
			return parseIPLiteral(host)
		}
	}

	return nil
}

func parseIPLiteral(raw string) net.IP {
	candidate := strings.TrimSpace(raw)
	candidate = strings.TrimPrefix(candidate, "[")
	candidate = strings.TrimSuffix(candidate, "]")
	if zoneIdx := strings.LastIndex(candidate, "%"); zoneIdx != -1 {
		candidate = candidate[:zoneIdx]
	}
	return net.ParseIP(candidate)
}
