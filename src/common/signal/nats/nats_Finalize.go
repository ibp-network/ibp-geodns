package nats

import (
	"encoding/json"
	"ibp-geodns/src/common/signal/types"
	"time"
)

func Finalize(pid types.ProposalID, finalStatus bool, state types.NodeState) {
	fm := types.FinalizeMessage{
		ProposalID:  pid,
		FinalStatus: finalStatus,
		DecidedAt:   time.Now().UTC(),
	}
	data, _ := json.Marshal(fm)
	Message(state.SubjectFinalize, data)
}
