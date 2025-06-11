package consensus

/*
   ---------------------------------------------------------------------------
   Snapshot‑based consensus manager
   ---------------------------------------------------------------------------

   – Each proposal carries the complete Snapshot (site/domain/endpoint).

   – A node votes YES if *its current local* snapshot is identical byte‑wise
     to the proposal. Otherwise it votes NO.

   – Quorum = ceil(2/3 × N).  Proposal passes when YES ≥ quorum, fails when
     NO ≥ quorum, or expires after timeout.

   – On PASS every node **must** promote the *proposal snapshot verbatim*
     via data.SetOfficialSnapshot(...).  Nobody writes local‑only data into
     the official store any more – this removes the divergence/flapping
     you observed.

   – The manager is self‑contained; it only needs an existing *nats.Conn*.
*/

import (
	"bytes"
	"encoding/json"
	"sync"
	"time"

	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"

	"github.com/nats-io/nats.go"
)

// ---------------------------------------------------------------------------
// wire format
// ---------------------------------------------------------------------------
const (
	msgTypeProposal = "proposal"
	msgTypeVote     = "vote"
	msgTypeClose    = "close"
)

type clusterMsg struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type proposalMsg struct {
	ID        string       `json:"id"`
	Proposer  string       `json:"proposer"`
	Timestamp time.Time    `json:"ts"`
	Snapshot  dat.Snapshot `json:"snapshot"`
}

type voteMsg struct {
	ID      string `json:"id"`
	Voter   string `json:"voter"`
	InFavor bool   `json:"in_favor"`
}

// ---------------------------------------------------------------------------
// in‑memory tracking
// ---------------------------------------------------------------------------
type proposal struct {
	msg       proposalMsg
	votes     map[string]bool
	yes, no   int
	expiresAt time.Time
	closed    bool
}

// ---------------------------------------------------------------------------
// Manager
// ---------------------------------------------------------------------------
type Manager struct {
	mu         sync.Mutex
	nodeID     string
	passQuorum int
	nc         *nats.Conn
	props      map[string]*proposal
	timeout    time.Duration
}

// NewManager starts the consensus engine and subscribes to "consensus.cluster".
func NewManager(nodeID string, nc *nats.Conn, monitorCount int) *Manager {
	if monitorCount < 1 {
		monitorCount = 1 // safe default until membership list settles
	}
	pass := (monitorCount*2 + 2) / 3 // ceil(2/3 × N)

	m := &Manager{
		nodeID:     nodeID,
		passQuorum: pass,
		nc:         nc,
		props:      make(map[string]*proposal),
		timeout:    45 * time.Second,
	}

	_, _ = nc.Subscribe("consensus.cluster", m.handleClusterMsg)
	return m
}

// ProposeSnapshot publishes a new proposal; exported for callers.
func (m *Manager) ProposeSnapshot(snap dat.Snapshot) {
	m.mu.Lock()
	defer m.mu.Unlock()

	id := m.nodeID + "-" + time.Now().UTC().Format("20060102T150405.000")
	p := proposalMsg{
		ID:        id,
		Proposer:  m.nodeID,
		Timestamp: time.Now().UTC(),
		Snapshot:  snap,
	}
	m.props[id] = &proposal{
		msg:       p,
		votes:     map[string]bool{m.nodeID: true}, // implicit YES
		yes:       1,
		no:        0,
		expiresAt: time.Now().Add(m.timeout),
	}

	m.broadcast(msgTypeProposal, p)
	m.broadcast(msgTypeVote, voteMsg{ID: id, Voter: m.nodeID, InFavor: true})
}

// ---------------------------------------------------------------------------
// internal helpers
// ---------------------------------------------------------------------------
func (m *Manager) broadcast(t string, payload interface{}) {
	raw, _ := json.Marshal(payload)
	wrap, _ := json.Marshal(clusterMsg{Type: t, Data: raw})
	_ = m.nc.Publish("consensus.cluster", wrap)
}

func (m *Manager) handleClusterMsg(msg *nats.Msg) {
	var wrapper clusterMsg
	if err := json.Unmarshal(msg.Data, &wrapper); err != nil {
		return
	}
	switch wrapper.Type {
	case msgTypeProposal:
		var p proposalMsg
		_ = json.Unmarshal(wrapper.Data, &p)
		m.onProposal(p)
	case msgTypeVote:
		var v voteMsg
		_ = json.Unmarshal(wrapper.Data, &v)
		m.onVote(v)
	case msgTypeClose:
		var id string
		_ = json.Unmarshal(wrapper.Data, &id)
		m.onClose(id)
	}
}

// ---------------------------------------------------------------------------
// PROPOSAL
// ---------------------------------------------------------------------------
func (m *Manager) onProposal(p proposalMsg) {
	m.mu.Lock()
	if _, exists := m.props[p.ID]; exists {
		m.mu.Unlock()
		return
	}
	m.props[p.ID] = &proposal{
		msg:       p,
		votes:     map[string]bool{p.Proposer: true},
		yes:       1,
		no:        0,
		expiresAt: p.Timestamp.Add(m.timeout),
	}
	m.mu.Unlock()

	local := dat.BuildSnapshot(dat.GetLocalResults())
	if snapshotsEqual(local, p.Snapshot) {
		m.broadcast(msgTypeVote, voteMsg{ID: p.ID, Voter: m.nodeID, InFavor: true})
	} else {
		m.broadcast(msgTypeVote, voteMsg{ID: p.ID, Voter: m.nodeID, InFavor: false})
	}
}

// ---------------------------------------------------------------------------
// VOTE
// ---------------------------------------------------------------------------
func (m *Manager) onVote(v voteMsg) {
	m.mu.Lock()
	defer m.mu.Unlock()

	prop, ok := m.props[v.ID]
	if !ok || prop.closed {
		return
	}
	if _, seen := prop.votes[v.Voter]; seen {
		return
	}
	prop.votes[v.Voter] = v.InFavor
	if v.InFavor {
		prop.yes++
	} else {
		prop.no++
	}
	m.evaluate(prop)
}

// ---------------------------------------------------------------------------
// CLOSE
// ---------------------------------------------------------------------------
func (m *Manager) onClose(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.props[id]; ok {
		p.closed = true
		delete(m.props, id)
	}
}

// ---------------------------------------------------------------------------
// evaluation logic  (manager.mu must already be held)
// ---------------------------------------------------------------------------
func (m *Manager) evaluate(p *proposal) {
	if p.yes >= m.passQuorum {
		dat.SetOfficialSnapshot(p.msg.Snapshot)
		log.Log(log.Info, "[Consensus] Proposal %s PASSED (YES=%d / NO=%d)", p.msg.ID, p.yes, p.no)
		p.closed = true
		m.broadcast(msgTypeClose, p.msg.ID)
		delete(m.props, p.msg.ID)
		return
	}
	if p.no >= m.passQuorum {
		log.Log(log.Info, "[Consensus] Proposal %s FAILED (YES=%d / NO=%d)", p.msg.ID, p.yes, p.no)
		p.closed = true
		m.broadcast(msgTypeClose, p.msg.ID)
		delete(m.props, p.msg.ID)
		return
	}
	if time.Now().After(p.expiresAt) {
		log.Log(log.Warn, "[Consensus] Proposal %s EXPIRED (YES=%d / NO=%d)", p.msg.ID, p.yes, p.no)
		p.closed = true
		m.broadcast(msgTypeClose, p.msg.ID)
		delete(m.props, p.msg.ID)
	}
}

// ---------------------------------------------------------------------------
// utils
// ---------------------------------------------------------------------------
func snapshotsEqual(a, b dat.Snapshot) bool {
	aj, _ := json.Marshal(a)
	bj, _ := json.Marshal(b)
	return bytes.Equal(aj, bj)
}
