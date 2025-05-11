package nats

import (
	"encoding/json"
	"time"
)

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
	err := Message(state.SubjectPropose, data)
	return pid, err
}
