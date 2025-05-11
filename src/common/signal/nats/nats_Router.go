package nats

import (
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
)

// checkLocalStatus returns this node's local perspective: true=online, false=offline.
func checkLocalStatus(checkType string, checkName string, memberName string, domainName string, endpoint string) (bool, bool) {
	switch checkType {
	case "site":
		found, status := dat.GetLocalSiteStatus(checkName, memberName)
		if !found {
			return false, false
		}
		return true, status

	case "domain":
		found, status := dat.GetLocalDomainStatus(checkName, memberName, domainName)
		if !found {
			return false, false
		}
		return true, status

	case "endpoint":
		found, status := dat.GetLocalEndpointStatus(checkName, memberName, endpoint)
		if !found {
			return false, false
		}
		return true, status

	default:
		log.Log(log.Warn, "checkLocalStatus: unknown checkType %s, defaulting to offline", checkType)
		return false, false
	}
}
