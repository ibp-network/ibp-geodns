package nats

import (
	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"

	"ibp-geodns/src/common/signal/types"
)

// applyOfficialChanges updates the official results once a vote passes.
// Call this after integrateFinalDecision sets the final status.
//
// proposal: the Proposal that reached consensus
// finalStatus: the boolean result of the vote (true=online, false=offline)
func applyOfficialChanges(proposal types.Proposal) {
	member, memberExists := findMemberByName(proposal.MemberName)
	if !memberExists {
		log.Log(log.Warn, "applyOfficialChanges: member %s not found", proposal.MemberName)
		return
	}

	check, checkExists := findCheckByName(proposal.CheckName, proposal.CheckType)
	if !checkExists {
		log.Log(log.Warn, "applyOfficialChanges: check %s not found", proposal.CheckName)
		return
	}

	var service cfg.Service
	if proposal.CheckType == "domain" || proposal.CheckType == "endpoint" {
		serv, servExists := findServiceForDomain(proposal.DomainName)
		if !servExists && proposal.CheckType == "domain" {
			log.Log(log.Warn, "applyOfficialChanges: service for domain %s not found", proposal.DomainName)
			return
		}
		service = serv
	}

	switch proposal.CheckType {
	case "site":
		log.Log(log.Info, "Updating official result from finalized proposal check: %s member: %s status: %t", check.Name, member.Details.Name, proposal.ProposedStatus)
		go dat.UpdateOfficialSiteResult(check, member, proposal.ProposedStatus, proposal.ErrorText, proposal.Data)
	case "domain":
		log.Log(log.Info, "Updating official result from finalized proposal check: %s member: %s status: %t domain: %s", check.Name, member.Details.Name, proposal.ProposedStatus, proposal.DomainName)
		go dat.UpdateOfficialDomainResult(check, member, service, proposal.DomainName, proposal.ProposedStatus, proposal.ErrorText, proposal.Data)
	case "endpoint":
		log.Log(log.Info, "Updating official result from finalized proposal check: %s member: %s status: %t domain: %s endpoint: %s", check.Name, member.Details.Name, proposal.ProposedStatus, proposal.DomainName, proposal.Endpoint)
		go dat.UpdateOfficialEndpointResult(check, member, service, proposal.DomainName, proposal.Endpoint, proposal.ProposedStatus, proposal.ErrorText, proposal.Data)
	}
}
