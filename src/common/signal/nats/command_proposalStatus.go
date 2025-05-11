package nats

import (
	log "ibp-geodns/src/common/logging"

	"ibp-geodns/src/common/signal/types"
)

func ProposeCheckStatus(state types.NodeState, checkType, checkName, memberName, domainName, endpoint string, status bool, errorText string, dataMap map[string]interface{}) bool {

	// Check for an existing active proposal with matching parameters
	state.Mu.RLock()
	for _, pt := range state.Proposals {
		prop := pt.Proposal
		if !pt.Finalized &&
			prop.CheckType == checkType &&
			prop.CheckName == checkName &&
			prop.MemberName == memberName &&
			prop.DomainName == domainName &&
			prop.Endpoint == endpoint &&
			prop.ProposedStatus == status {
			state.Mu.RUnlock()
			log.Log(log.Debug, "Propose skipped: Active proposal already exists for CheckType=%s, CheckName=%s, MemberName=%s", checkType, checkName, memberName)
			return false
		}
	}
	state.Mu.RUnlock()

	Propose(checkType, checkName, memberName, domainName, endpoint, status, errorText, dataMap)
	return true
}
