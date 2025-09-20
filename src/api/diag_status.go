package api

import (
	log "github.com/ibp-network/ibp-geodns-libs/logging"
)

func DumpDomainStatus() {
	ServiceRecords.mu.RLock()
	defer ServiceRecords.mu.RUnlock()

	for dom, sc := range ServiceRecords.Services {
		log.Log(log.Info, "[Status] Domain = %s  (members=%d)", dom, len(sc.Members))
		for mem := range sc.Members {
			v4 := IsMemberOnlineForDomainIPv4(dom, mem)
			v6 := IsMemberOnlineForDomainIPv6(dom, mem)

			state := "ONLINE"
			if !v4 && !v6 {
				state = "OFFLINE"
			} else if !v4 {
				state = "PARTIAL(v4 OFF)"
			} else if !v6 {
				state = "PARTIAL(v6 OFF)"
			}
			log.Log(log.Info, "  - %-16s %s", mem, state)
		}
	}
}
