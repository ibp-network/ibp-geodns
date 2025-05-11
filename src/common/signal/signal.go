package signal

import (
	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	"ibp-geodns/src/common/signal/helpers"
	"ibp-geodns/src/common/signal/nats"
	types "ibp-geodns/src/common/signal/types"
	"os"
	"time"
)

var (
	state types.NodeState
)

func Init() {
	c := cfg.GetConfig()
	con := c.Local.Signal

	state.NodeID = con.NodeID
	state.ThisNode = types.NodeInfo{
		NodeID: con.NodeID,
	}

	state.Proposals = make(map[types.ProposalID]*types.ProposalTracking)
	state.ClusterNodes = make(map[string]types.NodeInfo)
	state.ClusterNodes[state.NodeID] = state.ThisNode

	state.SubjectPropose = "consensus.propose"
	state.SubjectVote = "consensus.vote"
	state.SubjectFinalize = "consensus.finalize"
	state.SubjectCluster = "consensus.cluster"
	state.ProposalTimeout = 4 * time.Second
	state.NatsUrl = con.Url

	err := nats.Connect(state.NatsUrl)
	if err != nil {
		log.Log(log.Fatal, "Nats connection error: %+v", err)
		os.Exit(1)
	}

	err = nats.Subscribe()
	if err != nil {
		log.Log(log.Fatal, "Nats subscription error: %+v", err)
		os.Exit(1)
	}

	go helpers.StartProposalCleanup(state)
}

func Shutdown() {
	nats.Disconnect()
}
