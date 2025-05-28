package nats

import (
	"sync"
	"time"
)

// ProposalID is a unique identifier for a proposal.
type ProposalID string

// Proposal contains the required fields for a monitor voting change.
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

// Vote is a node's vote on a proposal.
type Vote struct {
	ProposalID ProposalID `json:"ProposalID"`
	NodeID     string     `json:"NodeID"`
	Agree      bool       `json:"Agree"`
	Timestamp  time.Time  `json:"Timestamp"`
}

// FinalizeMessage announces the final status of a proposal.
type FinalizeMessage struct {
	ProposalID  ProposalID `json:"ProposalID"`
	FinalStatus bool       `json:"FinalStatus"`
	DecidedAt   time.Time  `json:"DecidedAt"`
}

// ProposalTracking holds votes and final status for a single proposal.
type ProposalTracking struct {
	Proposal    Proposal
	Votes       map[string]bool
	Finalized   bool
	FinalStatus bool
	Timer       *time.Timer
}

// NodeInfo holds data about a cluster node.
type NodeInfo struct {
	NodeID        string `json:"NodeID"`
	PublicAddress string `json:"PublicAddress"`
	ListenAddress string `json:"ListenAddress"`
	ListenPort    string `json:"ListenPort"`
}

// NodeState holds global cluster info.
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

// State is the global NodeState for this node.
var State NodeState

// UsageRequest is a collator request for DNS usage data.
type UsageRequest struct {
	StartDate  string `json:"startDate"`
	EndDate    string `json:"endDate"`
	Domain     string `json:"domain"`
	MemberName string `json:"memberName"`
	Country    string `json:"country"`
}

// UsageResponse is returned by DNS APIs for usage data.
type UsageResponse struct {
	NodeID       string        `json:"nodeID"`
	UsageRecords []UsageRecord `json:"usageRecords"`
}

// UsageRecord holds usage info for a domain/member/country on a specific date.
type UsageRecord struct {
	Date        string `json:"date"`
	Domain      string `json:"domain"`
	MemberName  string `json:"memberName"`
	CountryCode string `json:"countryCode"`
	Hits        int    `json:"hits"`
}

// DowntimeRequest is a collator request for monitor downtime data.
type DowntimeRequest struct {
	StartTime  time.Time `json:"startTime"`
	EndTime    time.Time `json:"endTime"`
	MemberName string    `json:"memberName"`
}

// DowntimeResponse is returned by monitors for downtime events.
type DowntimeResponse struct {
	NodeID string          `json:"nodeID"`
	Events []DowntimeEvent `json:"events"`
}

// DowntimeEvent holds offline/online data for a member or domain.
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
