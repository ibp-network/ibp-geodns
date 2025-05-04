package consensus

import (
	"sync"
	"time"
)

// NodeInfo holds information about a cluster node.
type NodeInfo struct {
	NodeID        string `json:"NodeID"`
	PublicAddress string `json:"PublicAddress"`
	ListenAddress string `json:"ListenAddress"`
	ListenPort    string `json:"ListenPort"`
}

type ProposalID string

type Proposal struct {
	ID             ProposalID             `json:"id"`
	CheckType      string                 `json:"CheckType"`
	CheckName      string                 `json:"CheckName"`
	MemberName     string                 `json:"MemberName"`
	DomainName     string                 `json:"DomainName"`
	Endpoint       string                 `json:"Endpoint"`
	ProposedStatus bool                   `json:"ProposedStatus"`
	ErrorText      string                 `json:"ErrorText"`
	Data           map[string]interface{} `json:"Data"`
	Timestamp      time.Time              `json:"Timestamp"`
}

type Vote struct {
	ProposalID ProposalID `json:"ProposalID"`
	NodeID     string     `json:"NodeID"`
	Agree      bool       `json:"Agree"`
	Timestamp  time.Time  `json:"Timestamp"`
}

type FinalizeMessage struct {
	ProposalID  ProposalID `json:"ProposalID"`
	FinalStatus bool       `json:"FinalStatus"`
	DecidedAt   time.Time  `json:"DecidedAt"`
}

type ProposalTracking struct {
	Proposal    Proposal
	Votes       map[string]bool
	Finalized   bool
	FinalStatus bool
	Timer       *time.Timer
}

type NodeState struct {
	NodeID          string
	ThisNode        NodeInfo
	Mu              sync.RWMutex
	Proposals       map[ProposalID]*ProposalTracking
	ClusterNodes    map[string]NodeInfo // NodeID -> NodeInfo
	SubjectPropose  string
	SubjectVote     string
	SubjectFinalize string
	SubjectCluster  string
	ProposalTimeout time.Duration
	NatsUrl         string
	JoinUrl         string
}

var (
	state NodeState
)

type ProposalCaches struct {
	Cache map[string]bool
	Mu    sync.RWMutex
}
