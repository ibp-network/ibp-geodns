package main

/*
   IBP GeoDNS – Collator service
   -----------------------------
   • Consolidates proposals / votes / finalize messages coming from the
     monitoring & DNS layers.
   • Persists every change to MySQL for long‑term audit.
   • Periodically requests per‑DNS‑node usage statistics so that billing
     remains centralised.
   • Listens on the same NATS cluster as every other component.

   Build:  go build ./src/IBPCollator
   Run  :  ./IBPCollator -config /path/to/collator.json
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

	"ibp-geodns/src/common/config"
	"ibp-geodns/src/common/data2"
)

/* -------------------------------------------------------------------------- */
/*  Configuration                                                             */
/* -------------------------------------------------------------------------- */

// Config is loaded from the JSON file passed via -config.
type Config struct {
	Node string `json:"node"`

	MySQL struct {
		// <user>:<pass>@tcp(<host>:<port>)/<db>?parseTime=true
		DSN      string `json:"dsn"`
		MaxOpen  int    `json:"max_open"`
		MaxIdle  int    `json:"max_idle"`
		ConnLife int    `json:"conn_max_lifetime_seconds"`
	} `json:"mysql"`

	NATS struct {
		URL          string        `json:"url"`
		ReconnectDur time.Duration `json:"reconnect_wait_ms"`
	} `json:"nats"`
}

/* -------------------------------------------------------------------------- */
/*  Collator runtime structure                                                */
/* -------------------------------------------------------------------------- */

type collator struct {
	cfg   Config
	db    *sql.DB
	nc    *nats.Conn
	votes map[string]Vote // last vote received, keyed by member name
	mu    sync.Mutex      // protects votes
}

/* -------------------------------------------------------------------------- */
/*  NATS payloads                                                             */
/* -------------------------------------------------------------------------- */

// Vote is broadcast by IBPDns nodes when they vote on a proposal.
type Vote struct {
	Member   string `json:"member"`
	Proposal string `json:"proposal_id"`
	Value    bool   `json:"value"`
}

// Finalize is emitted by the chair node to close a voting round.
type Finalize struct {
	Proposal string `json:"proposal_id"`
}

// Proposal carries all metadata required to create a check row in MySQL.
type Proposal struct {
	ID        string    `json:"id"`
	IPv6      string    `json:"ipv6"`
	Domain    string    `json:"domain"`
	Member    string    `json:"member"`
	CheckName string    `json:"check_name"`
	CheckType string    `json:"check_type"`
	CreatedAt time.Time `json:"created_at"`
}

/* -------------------------------------------------------------------------- */
/*  main()                                                                    */
/* -------------------------------------------------------------------------- */

func main() {
	/* ---- CLI flags ------------------------------------------------------- */

	cfgPath := flag.String("config", "", "path to JSON configuration file")
	flag.Parse()
	if *cfgPath == "" {
		fmt.Fprintln(os.Stderr, "ERROR: -config flag is required")
		os.Exit(1)
	}

	/* ---- Load config ----------------------------------------------------- */

	var cfg Config
	if err := readJSONFile(*cfgPath, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "failed to read config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("IBP‑GeoDNS Collator v0.4.1 starting (node=%s)\n", cfg.Node)

	/* ---- MySQL ----------------------------------------------------------- */

	globalCfg := config.GetConfig() // reuse cluster‑wide JSON for credentials
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
		globalCfg.Local.Mysql.User,
		globalCfg.Local.Mysql.Pass,
		globalCfg.Local.Mysql.Host,
		globalCfg.Local.Mysql.Port,
		globalCfg.Local.Mysql.DB,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "MySQL connection failed: %v\n", err)
		os.Exit(1)
	}
	if err := db.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "unable to ping MySQL: %v\n", err)
		os.Exit(1)
	}
	data2.Init() // ensure data2.DB is initialised
	fmt.Println("[data2] Connected to MySQL")

	/* ---- NATS ------------------------------------------------------------ */

	nc, err := nats.Connect(
		cfg.NATS.URL,
		nats.Name(fmt.Sprintf("IBPCollator‑%s", cfg.Node)),
		nats.ReconnectWait(cfg.NATS.ReconnectDur*time.Millisecond),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot connect to NATS: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("[NATS] Connected (%s)\n", cfg.NATS.URL)

	/* ---- Bring the collator to life ------------------------------------- */

	col := &collator{
		cfg:   cfg,
		db:    db,
		nc:    nc,
		votes: make(map[string]Vote),
	}
	if err := col.subscribe(); err != nil {
		fmt.Fprintf(os.Stderr, "[collator] subscribe error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("[collator] subscriptions active")

	go col.startUsageCollector()

	/* ---- Graceful shutdown --------------------------------------------- */

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	fmt.Println("signal received – shutting down …")

	nc.Drain()
	db.Close()
	fmt.Println("bye")
}

/* -------------------------------------------------------------------------- */
/*  Subscription & handlers                                                   */
/* -------------------------------------------------------------------------- */

func (c *collator) subscribe() error {
	// Votes
	if _, err := c.nc.Subscribe("ibp.vote", c.handleVote); err != nil {
		return err
	}
	// Finalize round
	if _, err := c.nc.Subscribe("ibp.finalize", c.handleFinalize); err != nil {
		return err
	}
	// Proposals
	if _, err := c.nc.Subscribe("ibp.proposal", c.handleProposal); err != nil {
		return err
	}
	return nil
}

func (c *collator) handleVote(m *nats.Msg) {
	var v Vote
	if err := json.Unmarshal(m.Data, &v); err != nil {
		fmt.Printf("[collator] bad vote payload: %v\n", err)
		return
	}

	c.mu.Lock()
	c.votes[v.Member] = v
	c.mu.Unlock()
}

func (c *collator) handleFinalize(m *nats.Msg) {
	var f Finalize
	if err := json.Unmarshal(m.Data, &f); err != nil {
		fmt.Printf("[collator] bad finalize payload: %v\n", err)
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Tally votes for this proposal
	yes, total := 0, 0
	for _, v := range c.votes {
		if v.Proposal == f.Proposal {
			total++
			if v.Value {
				yes++
			}
		}
	}
	c.deleteVotesForProposal(f.Proposal)

	fmt.Printf("[collator] finalize %s – yes=%d / %d\n", f.Proposal, yes, total)
	if err := data2.MarkProposalFinal(f.Proposal, yes, total); err != nil {
		fmt.Printf("[collator] MarkProposalFinal: %v\n", err)
	}
}

func (c *collator) deleteVotesForProposal(pid string) {
	for k, v := range c.votes {
		if v.Proposal == pid {
			delete(c.votes, k)
		}
	}
}

func (c *collator) handleProposal(m *nats.Msg) {
	var p Proposal
	if err := json.Unmarshal(m.Data, &p); err != nil {
		fmt.Printf("[collator] bad proposal payload: %v\n", err)
		return
	}
	if p.CreatedAt.IsZero() {
		p.CreatedAt = time.Now().UTC()
	}

	if err := data2.StoreProposal(data2.Proposal(p)); err != nil {
		fmt.Printf("[collator] StoreProposal: %v\n", err)
	}
}

/* -------------------------------------------------------------------------- */
/*  Usage collector (periodic broadcast)                                      */
/* -------------------------------------------------------------------------- */

func (c *collator) startUsageCollector() {
	tick := time.NewTicker(30 * time.Minute)
	defer tick.Stop()

	for range tick.C {
		if err := c.nc.Publish("ibp.usage.request", nil); err != nil {
			fmt.Printf("[collator] usage request error: %v\n", err)
		} else {
			fmt.Println("[collator] requesting usage from all DNS nodes")
		}
	}
}

/* -------------------------------------------------------------------------- */
/*  Utility                                                                   */
/* -------------------------------------------------------------------------- */

func readJSONFile(path string, v any) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(v)
}
