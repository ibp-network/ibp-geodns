package monitor

/*
   check orchestration + *per‑result* consensus publishing

   • Every time we store a fresh local result we compare it to the **current
     official** status for that (check, member, v4/v6, etc.).
     – If the status differs, we broadcast a single lightweight proposal via
       nats.ProposeCheckStatus().
     – If it is identical we do nothing – no floods.

   • The old snapshot‑based code and SetConsensusManager() are gone; we do not
     import or reference IBPMonitor/consensus anywhere any more.

   • Quorum logic lives in src/common/nats/cmd_Consensus.go and now sees the
     *real* monitor count because every node broadcasts JOIN frames that carry
     a non‑empty NodeID (see roles.go).
*/

import (
	"net/url"
	"strings"
	"time"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
	natsCommon "ibp-geodns/src/common/nats"
)

// ---------------------------------------------------------------------------
// Registry of checks
// ---------------------------------------------------------------------------

var CheckRegistry = struct {
	Site     map[string]CheckSiteFunc
	Domain   map[string]CheckDomainFunc
	Endpoint map[string]CheckEndpointFunc
}{
	Site:     make(map[string]CheckSiteFunc),
	Domain:   make(map[string]CheckDomainFunc),
	Endpoint: make(map[string]CheckEndpointFunc),
}

// Function signatures.
type (
	CheckSiteFunc     func(check cfg.Check, member cfg.Member)
	CheckDomainFunc   func(check cfg.Check, domain string, service cfg.Service, member cfg.Member)
	CheckEndpointFunc func(check cfg.Check, endpoint string, service cfg.Service, member cfg.Member)
)

// ---------------------------------------------------------------------------
// registration helpers
// ---------------------------------------------------------------------------

func RegisterSiteCheck(name string, fn CheckSiteFunc) {
	CheckRegistry.Site[name] = fn
}

func RegisterDomainCheck(name string, fn CheckDomainFunc) {
	CheckRegistry.Domain[name] = fn
}

func RegisterEndpointCheck(name string, fn CheckEndpointFunc) {
	CheckRegistry.Endpoint[name] = fn
}

// ---------------------------------------------------------------------------
// start all periodic timers (called once from monitor.Init())
// ---------------------------------------------------------------------------

func startChecks() {
	go initSiteCheck()
	go initDomainCheck()
	go initEndpointCheck()
}

// ---------------------------------------------------------------------------
// SITE checks
// ---------------------------------------------------------------------------

func getSiteCheck(name string) (CheckSiteFunc, bool) {
	fn, ok := CheckRegistry.Site[name]
	return fn, ok
}

func initSiteCheck() {
	c := cfg.GetConfig()
	for _, ch := range c.Local.Checks {
		if ch.CheckType == "site" && ch.Enabled == 1 {
			if fn, ok := getSiteCheck(ch.Name); ok {
				go siteCheckTimer(ch, fn)
			}
		}
	}
}

func siteCheckTimer(ch cfg.Check, fn CheckSiteFunc) {
	time.Sleep(2 * time.Second)
	runSiteCheck(ch, fn)

	ticker := time.NewTicker(time.Duration(ch.CheckInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		runSiteCheck(ch, fn)
	}
}

func runSiteCheck(ch cfg.Check, fn CheckSiteFunc) {
	c := cfg.GetConfig()
	for _, m := range c.Members {
		if m.Service.Active == 1 && !m.Override {
			go func(check cfg.Check, mem cfg.Member) {
				defer func() {
					if r := recover(); r != nil {
						log.Log(log.Error, "SITE‑check %s on %s panicked: %v", check.Name, mem.Details.Name, r)
					}
				}()
				fn(check, mem)
			}(ch, m)
		}
	}
}

// UpdateSiteResultLocal is called by the concrete site‑check modules.
func UpdateSiteResultLocal(check cfg.Check, member cfg.Member, status bool, errText string,
	data map[string]interface{}, ipv6 bool) {

	dat.UpdateLocalSiteResult(check, member, status, errText, data, ipv6)
	proposeIfStatusChanged("site", check.Name, member.Details.Name, "", "",
		status, errText, data, ipv6)
}

// ---------------------------------------------------------------------------
// DOMAIN checks
// ---------------------------------------------------------------------------

func getDomainCheck(name string) (CheckDomainFunc, bool) {
	fn, ok := CheckRegistry.Domain[name]
	return fn, ok
}

func initDomainCheck() {
	c := cfg.GetConfig()
	for _, ch := range c.Local.Checks {
		if ch.CheckType == "domain" && ch.Enabled == 1 {
			if fn, ok := getDomainCheck(ch.Name); ok {
				go domainCheckTimer(ch, fn)
			}
		}
	}
}

func domainCheckTimer(ch cfg.Check, fn CheckDomainFunc) {
	time.Sleep(3 * time.Second)
	runDomainCheck(ch, fn)

	ticker := time.NewTicker(time.Duration(ch.CheckInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		runDomainCheck(ch, fn)
	}
}

func runDomainCheck(ch cfg.Check, fn CheckDomainFunc) {
	c := cfg.GetConfig()
	for svcName, svc := range c.Services {
		for _, mem := range c.Members {
			if mem.Service.Active == 1 && !mem.Override &&
				mem.Membership.Level >= svc.Configuration.LevelRequired {

				if assignedToService(svcName, mem) {
					doms := extractDomains(svc)
					for dom := range doms {
						go func(check cfg.Check, domain string, s cfg.Service, m cfg.Member) {
							defer func() {
								if r := recover(); r != nil {
									log.Log(log.Error, "DOMAIN‑check %s on %s crashed: %v", check.Name, m.Details.Name, r)
								}
							}()
							fn(check, domain, s, m)
						}(ch, dom, svc, mem)
						time.Sleep(100 * time.Millisecond)
					}
				}
			}
		}
	}
}

func UpdateDomainResultLocal(check cfg.Check, domain string, service cfg.Service,
	member cfg.Member, status bool, errText string, data map[string]interface{}, ipv6 bool) {

	dat.UpdateLocalDomainResult(check, member, service, domain, status, errText, data, ipv6)
	proposeIfStatusChanged("domain", check.Name, member.Details.Name, domain, "",
		status, errText, data, ipv6)
}

// ---------------------------------------------------------------------------
// ENDPOINT checks
// ---------------------------------------------------------------------------

func getEndpointCheck(name string) (CheckEndpointFunc, bool) {
	fn, ok := CheckRegistry.Endpoint[name]
	return fn, ok
}

func initEndpointCheck() {
	c := cfg.GetConfig()
	for _, ch := range c.Local.Checks {
		if ch.CheckType == "endpoint" && ch.Enabled == 1 {
			if fn, ok := getEndpointCheck(ch.Name); ok {
				go endpointCheckTimer(ch, fn)
			}
		}
	}
}

func endpointCheckTimer(ch cfg.Check, fn CheckEndpointFunc) {
	time.Sleep(4 * time.Second)
	runEndpointCheck(ch, fn)

	ticker := time.NewTicker(time.Duration(ch.CheckInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		runEndpointCheck(ch, fn)
	}
}

func runEndpointCheck(ch cfg.Check, fn CheckEndpointFunc) {
	c := cfg.GetConfig()
	for svcName, svc := range c.Services {
		for _, mem := range c.Members {
			if mem.Service.Active == 1 && !mem.Override &&
				mem.Membership.Level >= svc.Configuration.LevelRequired {

				if assignedToService(svcName, mem) {
					for _, prov := range svc.Providers {
						for _, rpc := range prov.RpcUrls {
							go func(check cfg.Check, endpoint string, s cfg.Service, m cfg.Member) {
								defer func() {
									if r := recover(); r != nil {
										log.Log(log.Error, "ENDPOINT‑check %s on %s crashed: %v", check.Name, m.Details.Name, r)
									}
								}()
								fn(check, endpoint, s, m)
							}(ch, rpc, svc, mem)
							time.Sleep(100 * time.Millisecond)
						}
					}
				}
			}
		}
	}
}

func UpdateEndpointResultLocal(check cfg.Check, member cfg.Member, service cfg.Service,
	endpoint string, status bool, errText string, data map[string]interface{}, ipv6 bool) {

	domain := parseUrlForDomain(endpoint)
	dat.UpdateLocalEndpointResult(check, member, service, domain, endpoint, status, errText, data, ipv6)
	proposeIfStatusChanged("endpoint", check.Name, member.Details.Name, domain, endpoint,
		status, errText, data, ipv6)
}

// ---------------------------------------------------------------------------
// consensus helper
// ---------------------------------------------------------------------------

// proposeIfStatusChanged compares the *new local* status with the official
// cluster status.  If they differ we broadcast a proposal.
func proposeIfStatusChanged(checkType, checkName, memberName, domainName, endpoint string,
	status bool, errText string, data map[string]interface{}, ipv6 bool) {

	var (
		found bool
		cur   bool
	)

	switch checkType {
	case "site":
		found, cur = dat.GetOfficialSiteStatus(checkName, memberName, ipv6)
	case "domain":
		found, cur = dat.GetOfficialDomainStatus(checkName, memberName, domainName, ipv6)
	case "endpoint":
		found, cur = dat.GetOfficialEndpointStatus(checkName, memberName, domainName, endpoint, ipv6)
	}

	if !found || cur != status {
		natsCommon.ProposeCheckStatus(
			checkType,
			checkName,
			memberName,
			domainName,
			endpoint,
			status,
			errText,
			data,
			ipv6,
		)
	}
}

// ---------------------------------------------------------------------------
// miscellany helpers
// ---------------------------------------------------------------------------

func assignedToService(svcName string, m cfg.Member) bool {
	for _, list := range m.ServiceAssignments {
		for _, v := range list {
			if v == svcName {
				return true
			}
		}
	}
	return false
}

func extractDomains(s cfg.Service) map[string]struct{} {
	out := make(map[string]struct{})
	for _, prov := range s.Providers {
		for _, rpc := range prov.RpcUrls {
			if d := parseUrlForDomain(rpc); d != "" {
				out[d] = struct{}{}
			}
		}
	}
	return out
}

// parseUrlForDomain normalises any RPC/WSS URL into host‑only (lower‑case) string.
func parseUrlForDomain(raw string) string {
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.ToLower(u.Hostname())
}
