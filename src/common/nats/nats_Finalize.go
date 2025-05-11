package nats

import (
	"encoding/json"
	"time"
)

func Finalize(pid ProposalID, finalStatus bool) {
	fm := FinalizeMessage{
		ProposalID:  pid,
		FinalStatus: finalStatus,
		DecidedAt:   time.Now().UTC(),
	}
	data, _ := json.Marshal(fm)
	Message(state.SubjectFinalize, data)
}
