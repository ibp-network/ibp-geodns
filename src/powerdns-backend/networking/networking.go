package networking

import (
	"ibp-geodns/networking/consensus"
	"ibp-geodns/networking/geoip"
	"ibp-geodns/networking/monitor"
	"ibp-geodns/networking/webserver"
)

func Init() {
	go geoip.Init()
	go consensus.Init()
	go monitor.Init()
	go webserver.Init()
}
