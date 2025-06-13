package nats

import (
	"encoding/json"

	data2 "ibp-geodns/src/common/data2"
	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

func cacheCollatorProposal(m *nats.Msg) {
	var p Proposal
	if err := json.Unmarshal(m.Data, &p); err != nil {
		log.Log(log.Error, "[collator] proposal unmarshal error: %v", err)
		return
	}

	data2.CacheProposal(data2.Proposal{
		ID:             string(p.ID),
		SenderNodeID:   p.SenderNodeID,
		CheckType:      p.CheckType,
		CheckName:      p.CheckName,
		MemberName:     p.MemberName,
		DomainName:     p.DomainName,
		Endpoint:       p.Endpoint,
		ProposedStatus: p.ProposedStatus,
		ErrorText:      p.ErrorText,
		Data:           p.Data,
		IsIPv6:         p.IsIPv6,
		Timestamp:      p.Timestamp,
		CreatedAt:      p.Timestamp,
	})

	log.Log(log.Debug, "[collator] cached proposal id=%s member=%s type=%s v6=%v",
		p.ID, p.MemberName, p.CheckType, p.IsIPv6)
}
