package data2

import (
	"sync"
	"time"
)

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

var State NodeState

type NodeInfo struct {
	NodeID        string    `json:"NodeID"`
	PublicAddress string    `json:"PublicAddress"`
	ListenAddress string    `json:"ListenAddress"`
	ListenPort    string    `json:"ListenPort"`
	NodeRole      string    `json:"NodeRole"`
	LastHeard     time.Time `json:"LastHeard"`
}

type ProposalID = string

type Proposal struct {
	ID             ProposalID             `json:"ID"`
	SenderNodeID   string                 `json:"SenderNodeID"`
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

	Domain    string    `json:"Domain,omitempty"`
	Member    string    `json:"Member,omitempty"`
	CreatedAt time.Time `json:"CreatedAt,omitempty"`
}

type ProposalTracking struct {
	Proposal  Proposal
	Votes     map[string]bool
	Finalized bool
	Passed    bool
	Timer     *time.Timer
}

type Vote struct {
	ProposalID   ProposalID `json:"ProposalID"`
	SenderNodeID string     `json:"SenderNodeID"`
	NodeID       string     `json:"NodeID"`
	Agree        bool       `json:"Agree"`
	Timestamp    time.Time  `json:"Timestamp"`
}

type FinalizeMessage struct {
	Proposal  Proposal  `json:"Proposal"`
	Passed    bool      `json:"Passed"`
	DecidedAt time.Time `json:"DecidedAt"`
}

type UsageRecord struct {
	Date        time.Time `json:"date"`
	NodeID      string    `json:"nodeID"`
	Domain      string    `json:"domain"`
	MemberName  string    `json:"memberName"`
	Asn         string    `json:"asn"`
	NetworkName string    `json:"networkName"`
	CountryCode string    `json:"countryCode"`
	CountryName string    `json:"countryName"`
	IsIPv6      bool      `json:"isIPv6"`
	Hits        int       `json:"hits"`
}

type UsageRequest struct {
	StartDate  string `json:"startDate"`
	EndDate    string `json:"endDate"`
	Domain     string `json:"domain"`
	MemberName string `json:"memberName"`
	Country    string `json:"country"`
}

type UsageResponse struct {
	NodeID       string        `json:"nodeID"`
	UsageRecords []UsageRecord `json:"usageRecords"`
	Error        string        `json:"error,omitempty"`
}

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

type DowntimeResponse struct {
	NodeID string          `json:"nodeID"`
	Events []DowntimeEvent `json:"events"`
	Error  string          `json:"error,omitempty"`
}

type ClusterMessage struct {
	Type    string     `json:"type"`
	Sender  NodeInfo   `json:"sender"`
	Members []NodeInfo `json:"members"`
}
