package main

/*  IBP‑GeoDNS ‑ Collator service
    --------------------------------
    Consolidates proposals / votes / finalise messages,
    gathers usage statistics from all DNS nodes and
    persists everything to MySQL for long‑term audit.
*/

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/nats-io/nats.go"

	cfg "ibp-geodns/src/common/config"
	data2 "ibp-geodns/src/common/data2"
	log "ibp-geodns/src/common/logging"
	natsCommon "ibp-geodns/src/common/nats"
)

var version = cfg.GetVersion()

/* -------------------------------------------------------------------------- */
/*  Collator runtime structure                                                */
/* -------------------------------------------------------------------------- */

type collator struct {
	cfg   cfg.Config
	db    *sql.DB
	votes map[string]natsCommon.Vote // keyed by NodeID (not member name any more)
	mu    sync.Mutex
}

/* -------------------------------------------------------------------------- */
/*  main()                                                                    */
/* -------------------------------------------------------------------------- */

func main() {
	log.Log(log.Info, "IBP‑GeoDNS Collator v%s starting …", version)

	cfgFile := flag.String("config", "ibpcollator.json", "Path to configuration file")
	flag.Parse()

	if _, err := os.Stat(*cfgFile); os.IsNotExist(err) {
		log.Log(log.Fatal, "configuration file not found: %s", *cfgFile)
		os.Exit(1)
	}

	/* ---- configuration & logging ------------------------------- */

	cfg.Init(*cfgFile)
	c := cfg.GetConfig()
	log.SetLogLevel(log.ParseLogLevel(c.Local.System.LogLevel))
	log.Log(log.Info, "collator log‑level: %s", c.Local.System.LogLevel)

	/* ---- MySQL -------------------------------------------------- */

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		c.Local.Mysql.User,
		c.Local.Mysql.Pass,
		c.Local.Mysql.Host,
		c.Local.Mysql.Port,
		c.Local.Mysql.DB,
	)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Log(log.Fatal, "MySQL connection failed: %v", err)
		os.Exit(1)
	}
	if err := db.Ping(); err != nil {
		log.Log(log.Fatal, "unable to ping MySQL: %v", err)
		os.Exit(1)
	}
	data2.Init()
	log.Log(log.Info, "[data2] connected to MySQL")

	/* ---- NATS --------------------------------------------------- */

	if err := natsCommon.Connect(); err != nil {
		log.Log(log.Fatal, "failed to connect to NATS: %v", err)
		os.Exit(1)
	}

	// Advertise ourselves to the cluster.
	natsCommon.State.NodeID = c.Local.Nats.NodeID
	natsCommon.State.ThisNode = natsCommon.NodeInfo{
		NodeID:        c.Local.Nats.NodeID,
		ListenAddress: "0.0.0.0",
		ListenPort:    "0",
		NodeRole:      "IBPCollator",
	}
	if err := natsCommon.EnableCollatorRole(); err != nil {
		log.Log(log.Fatal, "failed to enable Collator role: %v", err)
		os.Exit(1)
	}

	/* ---- bring the collator to life ---------------------------- */

	col := &collator{
		cfg:   c,
		db:    db,
		votes: make(map[string]natsCommon.Vote),
	}
	if err := col.subscribe(); err != nil {
		log.Log(log.Fatal, "subscription error: %v", err)
		os.Exit(1)
	}
	log.Log(log.Info, "[collator] NATS subscriptions active")

	go col.startUsageCollector()

	/* ---- graceful shutdown ------------------------------------- */

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Log(log.Info, "signal received – shutting down")
	_ = db.Close()
	log.Log(log.Info, "bye")
}

/* -------------------------------------------------------------------------- */
/*  NATS subscriptions & handlers                                             */
/* -------------------------------------------------------------------------- */

func (c *collator) subscribe() error {
	if _, err := natsCommon.Subscribe(natsCommon.State.SubjectPropose, c.handleProposal); err != nil {
		return err
	}
	if _, err := natsCommon.Subscribe(natsCommon.State.SubjectVote, c.handleVote); err != nil {
		return err
	}
	if _, err := natsCommon.Subscribe(natsCommon.State.SubjectFinalize, c.handleFinalize); err != nil {
		return err
	}
	if _, err := natsCommon.Subscribe("dns.usage.usageData", c.handleUsageData); err != nil {
		return err
	}
	return nil
}

/* ---------------------------- Proposal ------------------------------------ */

func (c *collator) handleProposal(m *nats.Msg) {
	var p natsCommon.Proposal
	if err := json.Unmarshal(m.Data, &p); err != nil {
		log.Log(log.Error, "[collator] bad proposal payload: %v", err)
		return
	}
	if p.Timestamp.IsZero() {
		p.Timestamp = time.Now().UTC()
	}

	if err := data2.StoreProposal(data2.Proposal{
		ID:        string(p.ID),
		IsIPv6:    p.IsIPv6,
		Domain:    p.DomainName,
		Member:    p.MemberName,
		CheckName: p.CheckName,
		CheckType: p.CheckType,
		CreatedAt: p.Timestamp,
	}); err != nil {
		log.Log(log.Error, "[collator] StoreProposal: %v", err)
	}
}

/* ------------------------------ Vote -------------------------------------- */

func (c *collator) handleVote(m *nats.Msg) {
	var v natsCommon.Vote
	if err := json.Unmarshal(m.Data, &v); err != nil {
		log.Log(log.Error, "[collator] bad vote payload: %v", err)
		return
	}

	key := v.NodeID
	if key == "" {
		key = v.SenderNodeID
	}
	if key == "" {
		// should never happen, but guard anyway
		return
	}

	c.mu.Lock()
	c.votes[key] = v
	c.mu.Unlock()
}

/* ---------------------------- Finalise ------------------------------------ */

func (c *collator) handleFinalize(m *nats.Msg) {
	var fm natsCommon.FinalizeMessage
	if err := json.Unmarshal(m.Data, &fm); err != nil {
		log.Log(log.Error, "[collator] bad finalize payload: %v", err)
		return
	}

	/* tally local vote cache for this proposal */
	c.mu.Lock()
	yes, total := 0, 0
	for _, v := range c.votes {
		if v.ProposalID == fm.Proposal.ID {
			total++
			if v.Agree {
				yes++
			}
			// delete to keep map size bounded
			delete(c.votes, v.NodeID)
		}
	}
	c.mu.Unlock()

	log.Log(log.Info, "[collator] FINALISE %s  yes=%d / %d  passed=%v",
		fm.Proposal.ID, yes, total, fm.Passed)

	if err := data2.MarkProposalFinal(string(fm.Proposal.ID), yes, total); err != nil {
		log.Log(log.Error, "[collator] MarkProposalFinal: %v", err)
	}
}

/* ---------------------------- Usage data ---------------------------------- */

func (c *collator) handleUsageData(m *nats.Msg) {
	var resp natsCommon.UsageResponse
	if err := json.Unmarshal(m.Data, &resp); err != nil {
		log.Log(log.Error, "[collator] usage‑data unmarshal error: %v", err)
		return
	}
	if len(resp.UsageRecords) == 0 {
		return
	}

	// ---- convert to data2.UsageRecord ---------------------------------
	records := make([]data2.UsageRecord, 0, len(resp.UsageRecords))
	for _, r := range resp.UsageRecords {
		// the DNS nodes send YYYY‑MM‑DD; convert to time.Time
		dt, err := time.Parse("2006-01-02", r.Date)
		if err != nil {
			log.Log(log.Warn, "[collator] invalid date %q skipped", r.Date)
			continue
		}
		records = append(records, data2.UsageRecord{
			Date:        dt,
			NodeID:      resp.NodeID, //  the DNS node that generated the slice
			Domain:      r.Domain,
			MemberName:  r.MemberName,
			Asn:         r.Asn,
			NetworkName: r.NetworkName,
			CountryCode: r.CountryCode,
			CountryName: r.CountryName,
			IsIPv6:      false, // current wire‑format has no v6 flag
			Hits:        r.Hits,
		})
	}
	// -------------------------------------------------------------------

	if err := data2.StoreUsageRecords(records); err != nil {
		log.Log(log.Error, "[collator] StoreUsageRecords: %v", err)
	}
}

/* -------------------------------------------------------------------------- */
/*  Usage collector (periodic request/aggregation)                            */
/* -------------------------------------------------------------------------- */

func (c *collator) startUsageCollector() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		<-ticker.C

		today := time.Now().UTC().Format("2006-01-02")
		req := data2.UsageRequest{
			StartDate:  today,
			EndDate:    today,
			Domain:     "",
			MemberName: "",
			Country:    "",
		}

		/*  ---- get usage from ALL DNS nodes via NATS ------------- */

		raw, err := natsCommon.RequestAllDnsUsage(req, 20*time.Second)
		if err != nil {
			log.Log(log.Error, "[collator] RequestAllDnsUsage: %v", err)
			continue
		}
		if len(raw) == 0 {
			log.Log(log.Info, "[collator] usage collector: 0 records returned")
			continue
		}

		/* ---- convert & store ----------------------------------- */

		var records []data2.UsageRecord
		for _, r := range raw {
			dt, err := time.Parse("2006-01-02", r.Date)
			if err != nil {
				continue
			}
			records = append(records, data2.UsageRecord{
				Date:        dt,
				NodeID:      natsCommon.State.NodeID, // aggregated slice; mark our node
				Domain:      r.Domain,
				MemberName:  r.MemberName,
				Asn:         r.Asn,
				NetworkName: r.NetworkName,
				CountryCode: r.CountryCode,
				CountryName: r.CountryName,
				IsIPv6:      false, // current wire‑format is v4 only
				Hits:        r.Hits,
			})
		}

		if err := data2.StoreUsageRecords(records); err != nil {
			log.Log(log.Error, "[collator] StoreUsageRecords: %v", err)
			continue
		}
		log.Log(log.Info, "[collator] stored %d DNS‑usage record(s)", len(records))
	}
}
