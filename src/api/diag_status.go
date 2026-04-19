package api

import (
	log "github.com/ibp-network/ibp-geodns-libs/logging"
)

func DumpDomainStatus() {
	ServiceRecords.mu.RLock()
	defer ServiceRecords.mu.RUnlock()

	for dom, sc := range ServiceRecords.Services {
		log.Log(log.Info, "[Status] Domain = %s  (members=%d)", dom, len(sc.Members))
		for memberKey, member := range sc.Members {
			monitorName := member.Details.Name
			if monitorName == "" {
				monitorName = memberKey
			}
			v4 := IsMemberOnlineForDomainIPv4(dom, monitorName)
			v6 := IsMemberOnlineForDomainIPv6(dom, monitorName)

			state := "ONLINE"
			if !v4 && !v6 {
				state = "OFFLINE"
			} else if !v4 {
				state = "PARTIAL(v4 OFF)"
			} else if !v6 {
				state = "PARTIAL(v6 OFF)"
			}
			log.Log(log.Info, "  - %-16s %s", monitorName, state)
		}
	}
}
