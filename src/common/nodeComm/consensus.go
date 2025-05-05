package nodeComm

import (
	"encoding/json"
	"os"
	"time"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"

	"github.com/google/uuid"
)

// Init loads config and sets up consensus system.
func Init() {
	c := cfg.GetConfig()
	con := c.System.Nats

	state.NodeID = con.NodeID
	state.ThisNode = NodeInfo{
		NodeID: con.NodeID,
	}

	state.Proposals = make(map[ProposalID]*ProposalTracking)
	state.ClusterNodes = make(map[string]NodeInfo)
	state.ClusterNodes[state.NodeID] = state.ThisNode

	state.SubjectPropose = "consensus.propose"
	state.SubjectVote = "consensus.vote"
	state.SubjectFinalize = "consensus.finalize"
	state.SubjectCluster = "consensus.cluster"
	state.ProposalTimeout = 4 * time.Second
	state.NatsUrl = con.Url

	err := ConnectNats(state.NatsUrl)
	if err != nil {
		log.Log(log.Debug, "Nats connection error: %+v", err)
		os.Exit(1)
	}

	err = subscribeSubjects()
	if err != nil {
		log.Log(log.Debug, "Nats subscription error: %+v", err)
		os.Exit(1)
	}

	go StartProposalCleanup()
}

func Shutdown() {
	CloseNats()
}

func Propose(checkType string, checkName string, memberName string, domainName string, endpoint string, proposedStatus bool, errorText string, dataMap map[string]interface{}) (ProposalID, error) {
	pid := ProposalID(generateProposalID())
	prop := Proposal{
		ID:             pid,
		CheckType:      checkType,
		CheckName:      checkName,
		MemberName:     memberName,
		DomainName:     domainName,
		Endpoint:       endpoint,
		ProposedStatus: proposedStatus,
		ErrorText:      errorText,
		Data:           dataMap,
		Timestamp:      time.Now().UTC(),
	}

	pt := &ProposalTracking{
		Proposal: prop,
		Votes:    make(map[string]bool),
	}

	state.Mu.Lock()
	state.Proposals[pid] = pt
	state.Mu.Unlock()

	pt.Timer = time.AfterFunc(state.ProposalTimeout, func() {
		finalizeVote(pid)
	})

	data, _ := json.Marshal(prop)
	err := publishMessage(state.SubjectPropose, data)
	return pid, err
}

func generateProposalID() string {
	return uuid.New().String()
}

func ProposeCheckStatus(checkType, checkName, memberName, domainName, endpoint string, status bool, errorText string, dataMap map[string]interface{}) bool {

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
			//log.Log(log.Debug, "Propose skipped: Active proposal already exists for CheckType=%s, CheckName=%s, MemberName=%s", checkType, checkName, memberName)
			return false
		}
	}
	state.Mu.RUnlock()

	Propose(checkType, checkName, memberName, domainName, endpoint, status, errorText, dataMap)
	return true
}

// applyOfficialChanges updates the official results once a vote passes.
// Call this after integrateFinalDecision sets the final status.
//
// proposal: the Proposal that reached consensus
// finalStatus: the boolean result of the vote (true=online, false=offline)
func applyOfficialChanges(proposal Proposal) {
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
		log.Log(log.Debug, "Updating official result from finalized proposal check: %s member: %s status: %t", check.Name, member.Details.Name, proposal.ProposedStatus)
		go dat.UpdateOfficialSiteResult(check, member, proposal.ProposedStatus, proposal.ErrorText, proposal.Data)
	case "domain":
		log.Log(log.Debug, "Updating official result from finalized proposal check: %s member: %s status: %t domain: %s", check.Name, member.Details.Name, proposal.ProposedStatus, proposal.DomainName)
		go dat.UpdateOfficialDomainResult(check, member, service, proposal.DomainName, proposal.ProposedStatus, proposal.ErrorText, proposal.Data)
	case "endpoint":
		log.Log(log.Debug, "Updating official result from finalized proposal check: %s member: %s status: %t domain: %s endpoint: %s", check.Name, member.Details.Name, proposal.ProposedStatus, proposal.DomainName, proposal.Endpoint)
		go dat.UpdateOfficialEndpointResult(check, member, service, proposal.DomainName, proposal.Endpoint, proposal.ProposedStatus, proposal.ErrorText, proposal.Data)
	}
}
