package api

import (
	log "ibp-geodns/src/common/logging"
)

// DumpDomainStatus prints current online / offline state per domain & member.
// It is called from the monitor‑poller after every refresh so operators can
// immediately verify that OFFLINE results are honoured.
func DumpDomainStatus() {
	ServiceRecords.mu.RLock()
	defer ServiceRecords.mu.RUnlock()

	for domain, svc := range ServiceRecords.Services {
		log.Log(log.Info, "[Status] Domain = %s  (members=%d)", domain, len(svc.Members))
		for memberName := range svc.Members {
			on4 := IsMemberOnlineForDomainIPv4(domain, memberName)
			on6 := IsMemberOnlineForDomainIPv6(domain, memberName)

			// if a member lacks both IPs we simply note it
			state := "ONLINE"
			if !on4 && !on6 {
				state = "OFFLINE"
			} else if !on4 {
				state = "PARTIAL(v4 OFF)"
			} else if !on6 {
				state = "PARTIAL(v6 OFF)"
			}
			log.Log(log.Info, "  - %-16s %s", memberName, state)
		}
	}
}
