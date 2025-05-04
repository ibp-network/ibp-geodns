package networking

import (
	"ibp-geodns/src/common/geoip"
	"ibp-geodns/src/pdnsBackend/networking/consensus"
	"ibp-geodns/src/pdnsBackend/networking/monitor"
	"ibp-geodns/src/pdnsBackend/networking/webserver"
)

func Init() {
	go geoip.Init()
	go consensus.Init()
	go monitor.Init()
	go webserver.Init()
}
