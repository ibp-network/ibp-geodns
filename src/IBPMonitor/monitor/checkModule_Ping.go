package monitor

import (
	"fmt"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"

	"github.com/go-ping/ping"
)

func init() {
	RegisterSiteCheck("ping", PingCheck)
}

func PingCheck(check cfg.Check, member cfg.Member) {
	if member.Service.ServiceIPv4 != "" {
		runPingSingle(check, member, false)
	}

	if member.Service.ServiceIPv6 != "" {
		runPingSingle(check, member, true)
	}
}

func runPingSingle(check cfg.Check, member cfg.Member, isIPv6 bool) {
	var ipToPing string
	if isIPv6 {
		ipToPing = member.Service.ServiceIPv6
	} else {
		ipToPing = member.Service.ServiceIPv4
	}

	pingCount := getIntOption(check.ExtraOptions, "PingCount", 3)
	pingInterval := time.Duration(getIntOption(check.ExtraOptions, "PingInterval", 100)) * time.Millisecond
	pingTimeout := time.Duration(getIntOption(check.ExtraOptions, "PingTimeout", 1000)) * time.Millisecond
	pingSize := getIntOption(check.ExtraOptions, "PingSize", 32)
	pingTTL := getIntOption(check.ExtraOptions, "PingTTL", 64)
	maxPacketLoss := getFloatOption(check.ExtraOptions, "MaxPacketLoss", 5.0)
	maxLatency := int64(getIntOption(check.ExtraOptions, "MaxLatency", 800))

	pinger, err := ping.NewPinger(ipToPing)
	if err != nil {
		UpdateSiteResultLocal(check, member, false,
			fmt.Sprintf("Ping error init: %v", err),
			nil,
			isIPv6,
		)
		return
	}
	if isIPv6 {
		pinger.SetNetwork("ip6")
	}

	pinger.Count = pingCount
	pinger.Interval = pingInterval
	pinger.Timeout = pingTimeout * time.Duration(pingCount)
	pinger.Size = pingSize
	pinger.TTL = pingTTL
	pinger.SetPrivileged(true)

	err = pinger.Run()
	if err != nil {
		UpdateSiteResultLocal(check, member, false, err.Error(), nil, isIPv6)
		return
	}
	stats := pinger.Statistics()

	success := stats.PacketsRecv > 0 && stats.PacketLoss <= maxPacketLoss && stats.AvgRtt.Milliseconds() <= maxLatency
	var msg string
	if !success {
		msg = fmt.Sprintf("PingCheck: avgRtt=%dms, loss=%.0f%%", stats.AvgRtt.Milliseconds(), stats.PacketLoss)
	}
	dataMap := map[string]interface{}{
		"PacketLoss": stats.PacketLoss,
		"MinRtt":     stats.MinRtt.Milliseconds(),
		"AvgRtt":     stats.AvgRtt.Milliseconds(),
		"MaxRtt":     stats.MaxRtt.Milliseconds(),
		"StdDevRtt":  stats.StdDevRtt.Milliseconds(),
	}

	UpdateSiteResultLocal(check, member, success, msg, dataMap, isIPv6)
	log.Log(log.Debug, "Ping check completed for %s isIPv6=%v success=%v", member.Details.Name, isIPv6, success)
}
