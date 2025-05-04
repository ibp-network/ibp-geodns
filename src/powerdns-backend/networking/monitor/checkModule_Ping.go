package monitor

import (
	"fmt"
	"time"

	"common/config"

	"github.com/go-ping/ping"
)

func init() {
	RegisterSiteCheck("ping", PingCheck)
}

func PingCheck(check config.Check, member config.Member) {
	// Prepare ping parameters
	pingCount := getIntOption(check.ExtraOptions, "PingCount", 3)
	pingInterval := time.Duration(getIntOption(check.ExtraOptions, "PingInterval", 100)) * time.Millisecond
	pingTimeout := time.Duration(getIntOption(check.ExtraOptions, "PingTimeout", 1000)) * time.Millisecond
	pingSize := getIntOption(check.ExtraOptions, "PingSize", 32)
	pingTTL := getIntOption(check.ExtraOptions, "PingTTL", 64)
	maxPacketLoss := getFloatOption(check.ExtraOptions, "MaxPacketLoss", 5.0)
	maxLatency := int64(getIntOption(check.ExtraOptions, "MaxLatency", 800))

	pinger, err := ping.NewPinger(member.Service.ServiceIPv4)
	if err != nil {
		go UpdateSiteResultLocal(check, member, false, err.Error(), nil)
		return
	}

	pinger.Count = pingCount
	pinger.Interval = pingInterval
	pinger.Timeout = pingTimeout * time.Duration(pingCount)
	pinger.Size = pingSize
	pinger.TTL = pingTTL
	pinger.SetPrivileged(true)

	err = pinger.Run()
	if err != nil {
		go UpdateSiteResultLocal(check, member, false, err.Error(), nil)
		return
	}

	stats := pinger.Statistics()

	// Process statistics
	success := stats.PacketsRecv > 0 && stats.PacketLoss <= maxPacketLoss && stats.AvgRtt.Milliseconds() <= maxLatency

	var msg string
	if !success {
		msg = fmt.Sprintf("Error: Average RTT latency of '%d'ms and packet loss '%.0f%%'", stats.AvgRtt.Milliseconds(), stats.PacketLoss)
	} else {
		msg = ""
	}

	go UpdateSiteResultLocal(
		check,
		member,
		success,
		msg,
		map[string]interface{}{
			"PacketLoss": stats.PacketLoss,
			"MinRtt":     stats.MinRtt.Milliseconds(),
			"AvgRtt":     stats.AvgRtt.Milliseconds(),
			"MaxRtt":     stats.MaxRtt.Milliseconds(),
			"StdDevRtt":  stats.StdDevRtt.Milliseconds(),
		},
	)
}
