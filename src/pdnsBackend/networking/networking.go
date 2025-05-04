package networking

import (
	"ibp-geodns/src/pdnsBackend/networking/monitor"
	"ibp-geodns/src/pdnsBackend/networking/webserver"
)

func Init() {
	go monitor.Init()
	go webserver.Init()
}
