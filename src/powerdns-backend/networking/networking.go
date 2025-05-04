package networking

import (
	"ibp-geodns/src/common/networking/geoip"
	"ibp-geodns/src/powerdns-backend/networking/consensus"
	"ibp-geodns/src/powerdns-backend/networking/monitor"
	"ibp-geodns/src/powerdns-backend/networking/webserver"
)

func Init() {
	go geoip.Init()
	go consensus.Init()
	go monitor.Init()
	go webserver.Init()
}
