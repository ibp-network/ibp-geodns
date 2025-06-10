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
	NodeID        string    `json:"NodeID"`
	PublicAddress string    `json:"PublicAddress"`
	ListenAddress string    `json:"ListenAddress"`
	ListenPort    string    `json:"ListenPort"`
	NodeRole      string    `json:"NodeRole"` // "IBPMonitor", "IBPDns", "IBPCollator", etc.
	LastHeard     time.Time // Tracks last time we heard from this node
}

// Monitor Voting
type ProposalID string

type Proposal struct {
	ID             ProposalID             `json:"id"`
	SenderNodeID   string                 `json:"SenderNodeID"` // Added field to track which node initiated
	CheckType      string                 `json:"CheckType"`
	CheckName      string                 `json:"CheckName"`
	MemberName     string                 `json:"MemberName"`
	DomainName     string                 `json:"DomainName"`
	Endpoint       string                 `json:"Endpoint"`
	ProposedStatus bool                   `json:"ProposedStatus"`
	ErrorText      string                 `json:"ErrorText"`
	Data           map[string]interface{} `json:"Data"`
	IsIPv6         bool                   `json:"IsIPv6"`
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
	ProposalID   ProposalID `json:"ProposalID"`
	SenderNodeID string     `json:"SenderNodeID"` // Added to identify who is voting
	NodeID       string     `json:"NodeID"`
	Agree        bool       `json:"Agree"`
	Timestamp    time.Time  `json:"Timestamp"`
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
	Asn         string `json:"asn"`
	NetworkName string `json:"networkName"`
	CountryName string `json:"countryName"`
	Hits        int    `json:"hits"`
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
	IsIPv6     bool                   `json:"isIPv6"`
}

// Cluster membership messages
type ClusterMessage struct {
	Type    string     `json:"type"` // "join", "membership"
	Sender  NodeInfo   `json:"sender"`
	Members []NodeInfo `json:"members"` // populated for "membership" broadcasts
}

// Update DowntimeResponse to include Error field
type DowntimeResponse struct {
	NodeID string          `json:"nodeID"`
	Events []DowntimeEvent `json:"events"`
	Error  string          `json:"error,omitempty"`
}

// Update UsageResponse to include Error field
type UsageResponse struct {
	NodeID       string        `json:"nodeID"`
	UsageRecords []UsageRecord `json:"usageRecords"`
	Error        string        `json:"error,omitempty"`
}
