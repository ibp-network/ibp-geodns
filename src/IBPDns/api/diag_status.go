package api

import (
	log "ibp-geodns/src/common/logging"
)

// DumpDomainStatus prints ONLINE/OFFLINE per (domain,member) after each poll.
// This is purely operational‑diagnostic and does not affect runtime logic.
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
