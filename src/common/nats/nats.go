package nats

import (
	cfg "ibp-geodns/src/common/config"
	"os"
	"sync"
	"time"

	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

var (
	nc     *nats.Conn
	natsMu sync.Mutex
	state  NodeState
)

func Init() {
	c := cfg.GetConfig()
	con := c.Local.Signal

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

	err := Connect()
	if err != nil {
		log.Log(log.Fatal, "Nats connection error: %+v", err)
		os.Exit(1)
	}

	err = Subscribe()
	if err != nil {
		log.Log(log.Fatal, "Nats subscription error: %+v", err)
		os.Exit(1)
	}

	go StartGarbageCollection()
}
