package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	nc "ibp-geodns/src/common/nats"

	d2 "ibp-geodns/src/common/data2"

	"github.com/nats-io/nats.go"
)

var version = cfg.GetVersion()

// ---------- main ----------
func main() {
	fmt.Printf("IBP‑GeoDNS Collator v%s starting\n", version)

	confPath := flag.String("config", "ibpcollator.json", "Path to config file")
	flag.Parse()

	if _, err := os.Stat(*confPath); os.IsNotExist(err) {
		log.Log(log.Fatal, "[collator] config file missing: %s", *confPath)
	}

	cfg.Init(*confPath)
	c := cfg.GetConfig()
	log.SetLogLevel(log.ParseLogLevel(c.Local.System.LogLevel))

	// 1. MySQL (blocking)
	d2.Init()

	// 2. NATS
	if err := nc.Connect(); err != nil {
		log.Log(log.Fatal, "[collator] NATS connect error: %v", err)
	}
	defer nc.Disconnect()

	nc.State.NodeID = c.Local.Nats.NodeID
	nc.State.ThisNode = nc.NodeInfo{
		NodeID:        c.Local.Nats.NodeID,
		PublicAddress: "",
		ListenAddress: "0.0.0.0",
		ListenPort:    "0",
		NodeRole:      "IBPCollator",
	}
	if err := nc.EnableCollatorRole(); err != nil {
		log.Log(log.Fatal, "[collator] enable role error: %v", err)
	}

	// 3. Consensus listeners
	initConsensusListeners()

	// 4. Schedule usage polling
	go startUsageCollector()

	// Block forever
	select {}
}

// ---------- consensus wiring ----------
func initConsensusListeners() {
	// votes
	_, err := nc.Subscribe(nc.State.SubjectVote, handleVote)
	if err != nil {
		log.Log(log.Fatal, "[collator] subscribe vote: %v", err)
	}

	// final decisions
	_, err = nc.Subscribe(nc.State.SubjectFinalize, handleFinalize)
	if err != nil {
		log.Log(log.Fatal, "[collator] subscribe finalize: %v", err)
	}
	log.Log(log.Info, "[collator] subscribed to vote & finalize subjects")
}

// In‑memory vote cache  (ProposalID -> map[nodeID]bool)
var voteCache = make(map[nc.ProposalID]map[string]bool)

// ---------- handlers ----------
func handleVote(m *nats.Msg) {
	var v nc.Vote
	if err := json.Unmarshal(m.Data, &v); err != nil {
		log.Log(log.Error, "[collator] vote unmarshal: %v", err)
		return
	}
	if voteCache[v.ProposalID] == nil {
		voteCache[v.ProposalID] = make(map[string]bool)
	}
	voteCache[v.ProposalID][v.NodeID] = v.Agree
}

func handleFinalize(m *nats.Msg) {
	var fm nc.FinalizeMessage
	if err := json.Unmarshal(m.Data, &fm); err != nil {
		log.Log(log.Error, "[collator] finalize unmarshal: %v", err)
		return
	}

	prop := fm.Proposal
	votes := voteCache[prop.ID]
	delete(voteCache, prop.ID) // free memory

	// Build record
	rec := d2.NetStatusRecord{
		CheckType: checkTypeToInt(prop.CheckType),
		CheckName: prop.CheckName,
		CheckURL:  deriveCheckURL(prop),
		Domain:    prop.DomainName,
		Member:    prop.MemberName,
		Status:    prop.ProposedStatus,
		IsIPv6:    prop.IsIPv6,
		StartTime: time.Now().UTC(),
		VoteData:  votes,
	}

	if prop.ProposedStatus {
		// member came ONLINE – close any open downtime
		if err := d2.CloseOpenEvent(rec); err != nil {
			log.Log(log.Error, "[collator] close open event: %v", err)
		}
	} else {
		// OFFLINE – open / update event
		if err := d2.InsertNetStatus(rec); err != nil {
			log.Log(log.Error, "[collator] insert netStatus: %v", err)
		}
	}
}

// ---------- usage collection ----------
func startUsageCollector() {
	// first run shortly after boot
	time.Sleep(20 * time.Second)
	runUsageCollection()

	ticker := time.NewTicker(4 * time.Hour)
	for range ticker.C {
		runUsageCollection()
	}
}

func runUsageCollection() {
	log.Log(log.Info, "[collator] requesting usage from all DNS nodes")
	today := time.Now().UTC().Format("2006-01-02")
	req := nc.UsageRequest{
		StartDate:  today,
		EndDate:    today,
		Domain:     "",
		MemberName: "",
		Country:    "",
	}

	records, err := nc.RequestAllDnsUsage(req, 10*time.Second)
	if err != nil {
		log.Log(log.Error, "[collator] usage request error: %v", err)
		return
	}
	for _, r := range records {
		u := d2.UsageRecord{
			Date:        parseDate(r.Date),
			NodeID:      nc.State.NodeID,
			Domain:      r.Domain,
			MemberName:  r.MemberName,
			Asn:         r.Asn,
			NetworkName: r.NetworkName,
			CountryCode: r.CountryCode,
			CountryName: r.CountryName,
			IsIPv6:      false, // TODO: extend UsageRecord to transport v6/v4 info
			Hits:        r.Hits,
		}
		if err := d2.UpsertUsage(u); err != nil {
			log.Log(log.Error, "[collator] usage upsert: %v", err)
		}
	}
	log.Log(log.Info, "[collator] usage collection finished – %d rows", len(records))
}

// ---------- helpers ----------
func parseDate(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func checkTypeToInt(s string) int {
	switch strings.ToLower(s) {
	case "site":
		return 0
	case "domain":
		return 1
	case "endpoint":
		return 2
	default:
		return 9
	}
}

func deriveCheckURL(p nc.Proposal) string {
	if p.CheckType == "endpoint" {
		return p.Endpoint
	}
	return p.DomainName
}
