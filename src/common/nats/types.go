package nats

import (
	"sync"
	"time"
)

// NodeState holds global cluster info for the node
type NodeState struct {
	NodeID          string
	ThisNode        NodeInfo
	Mu              sync.RWMutex
	Proposals       map[ProposalID]*ProposalTracking
	ClusterNodes    map[string]NodeInfo
	SubjectPropose  string
	SubjectVote     string
	SubjectFinalize string
	SubjectCluster  string
	ProposalTimeout time.Duration
	NatsUrl         string
	JoinUrl         string
}

// State is the global NodeState instance
var State NodeState

// NodeInfo represents another node in the cluster
type NodeInfo struct {
	NodeID        string `json:"NodeID"`
	PublicAddress string `json:"PublicAddress"`
	ListenAddress string `json:"ListenAddress"`
	ListenPort    string `json:"ListenPort"`
}

// Monitor Voting
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

type ProposalTracking struct {
	Proposal    Proposal
	Votes       map[string]bool
	Finalized   bool
	FinalStatus bool
	Timer       *time.Timer
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

// DNS Usage
type UsageRequest struct {
	StartDate  string `json:"startDate"`
	EndDate    string `json:"endDate"`
	Domain     string `json:"domain"`
	MemberName string `json:"memberName"`
	Country    string `json:"country"`
}

type UsageRecord struct {
	Date        string `json:"date"`
	Domain      string `json:"domain"`
	MemberName  string `json:"memberName"`
	CountryCode string `json:"countryCode"`
	Hits        int    `json:"hits"`
}

type UsageResponse struct {
	NodeID       string        `json:"nodeID"`
	UsageRecords []UsageRecord `json:"usageRecords"`
}

// Monitor Stats / Downtime
type DowntimeRequest struct {
	StartTime  time.Time `json:"startTime"`
	EndTime    time.Time `json:"endTime"`
	MemberName string    `json:"memberName"`
}

type DowntimeEvent struct {
	MemberName string                 `json:"memberName"`
	CheckType  string                 `json:"checkType"`
	CheckName  string                 `json:"checkName"`
	DomainName string                 `json:"domainName,omitempty"`
	Endpoint   string                 `json:"endpoint,omitempty"`
	Status     bool                   `json:"status"`
	StartTime  time.Time              `json:"startTime"`
	EndTime    time.Time              `json:"endTime"`
	ErrorText  string                 `json:"errorText"`
	Data       map[string]interface{} `json:"data"`
}

type DowntimeResponse struct {
	NodeID string          `json:"nodeID"`
	Events []DowntimeEvent `json:"events"`
}
