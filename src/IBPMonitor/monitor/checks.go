package monitor

import (
	"net/url"
	"strings"
	"sync"
	"time"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
	natsCommon "ibp-geodns/src/common/nats"
)

// ----------------------------------------------------------------------
// Registry of checks
// ----------------------------------------------------------------------

var (
	CheckRegistry = struct {
		Site     map[string]CheckSiteFunc
		Domain   map[string]CheckDomainFunc
		Endpoint map[string]CheckEndpointFunc
		Mu       sync.RWMutex
	}{
		Site:     make(map[string]CheckSiteFunc),
		Domain:   make(map[string]CheckDomainFunc),
		Endpoint: make(map[string]CheckEndpointFunc),
	}
)

// Function signatures
type (
	CheckSiteFunc     func(check cfg.Check, member cfg.Member)
	CheckDomainFunc   func(check cfg.Check, domain string, service cfg.Service, member cfg.Member)
	CheckEndpointFunc func(check cfg.Check, endpoint string, service cfg.Service, member cfg.Member)
)

// ----------------------------------------------------------------------
// Registration helpers
// ----------------------------------------------------------------------

func RegisterSiteCheck(name string, fn CheckSiteFunc) {
	CheckRegistry.Mu.Lock()
	defer CheckRegistry.Mu.Unlock()
	CheckRegistry.Site[name] = fn
	log.Log(log.Info, "Registered site check '%s'", name)
}

func RegisterDomainCheck(name string, fn CheckDomainFunc) {
	CheckRegistry.Mu.Lock()
	defer CheckRegistry.Mu.Unlock()
	CheckRegistry.Domain[name] = fn
	log.Log(log.Info, "Registered domain check '%s'", name)
}

func RegisterEndpointCheck(name string, fn CheckEndpointFunc) {
	CheckRegistry.Mu.Lock()
	defer CheckRegistry.Mu.Unlock()
	CheckRegistry.Endpoint[name] = fn
}

// ----------------------------------------------------------------------
// Entry‑point – launched from monitor.Init()
// ----------------------------------------------------------------------

func startChecks() {
	go initSiteCheck()
	go initDomainCheck()
	go initEndpointCheck()
}

// ------------------------------------------------------------------
// SITE checks
// ------------------------------------------------------------------

func getSiteCheck(name string) (CheckSiteFunc, bool) {
	CheckRegistry.Mu.RLock()
	defer CheckRegistry.Mu.RUnlock()
	fn, ok := CheckRegistry.Site[name]
	return fn, ok
}

func initSiteCheck() {
	c := cfg.GetConfig()
	var siteChecks []cfg.Check
	for _, ch := range c.Local.Checks {
		if ch.CheckType == "site" && ch.Enabled == 1 {
			siteChecks = append(siteChecks, ch)
		}
	}

	for _, check := range siteChecks {
		if fn, exists := getSiteCheck(check.Name); exists {
			go siteCheckTimer(check, fn)
		}
	}
}

func siteCheckTimer(check cfg.Check, fn CheckSiteFunc) {
	time.Sleep(2 * time.Second)
	runSiteCheck(check, fn)

	ticker := time.NewTicker(time.Duration(check.CheckInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		runSiteCheck(check, fn)
	}
}

func runSiteCheck(check cfg.Check, fn CheckSiteFunc) {
	c := cfg.GetConfig()

	for _, member := range c.Members {
		if member.Service.Active == 1 && !member.Override {
			go func(ch cfg.Check, m cfg.Member) {
				done := make(chan struct{})
				timer := time.NewTimer(time.Duration(ch.Timeout) * time.Second)

				ipv4Exists := m.Service.ServiceIPv4 != ""
				ipv6Exists := m.Service.ServiceIPv6 != ""

				// Run the actual check
				go func() {
					defer func() {
						if r := recover(); r != nil {
							log.Log(log.Error, "Check %s for member %s panicked: %v", ch.Name, m.Details.Name, r)
							if ipv4Exists {
								UpdateSiteResultLocal(ch, m, false, "Check crashed", nil, false)
							}
							if ipv6Exists {
								UpdateSiteResultLocal(ch, m, false, "Check crashed", nil, true)
							}
						}
						close(done)
					}()
					fn(ch, m)
				}()

				select {
				case <-done:
				case <-timer.C:
					if ipv4Exists {
						UpdateSiteResultLocal(ch, m, false, "Check timed out", nil, false)
					}
					if ipv6Exists {
						UpdateSiteResultLocal(ch, m, false, "Check timed out", nil, true)
					}
				}
			}(check, member)
		}
	}
}

func UpdateSiteResultLocal(check cfg.Check, member cfg.Member, status bool, errorMsg string, dataMap map[string]interface{}, isIPv6 bool) {
	// 1) cache locally
	dat.UpdateLocalSiteResult(check, member, status, errorMsg, dataMap, isIPv6)

	// 2) compare with official snapshot
	found, officialStatus := dat.GetOfficialSiteStatus(check.Name, member.Details.Name, isIPv6)
	if !found {
		natsCommon.ProposeCheckStatus("site", check.Name, member.Details.Name, "", "", status, errorMsg, dataMap, isIPv6)
		return
	}
	if officialStatus != status {
		natsCommon.ProposeCheckStatus("site", check.Name, member.Details.Name, "", "", status, errorMsg, dataMap, isIPv6)
	}
}

// ------------------------------------------------------------------
// DOMAIN checks
// ------------------------------------------------------------------

func getDomainCheck(name string) (CheckDomainFunc, bool) {
	CheckRegistry.Mu.RLock()
	defer CheckRegistry.Mu.RUnlock()
	fn, ok := CheckRegistry.Domain[name]
	return fn, ok
}

func initDomainCheck() {
	c := cfg.GetConfig()
	var domainChecks []cfg.Check
	for _, ch := range c.Local.Checks {
		if ch.CheckType == "domain" && ch.Enabled == 1 {
			domainChecks = append(domainChecks, ch)
		}
	}
	for _, check := range domainChecks {
		if fn, exists := getDomainCheck(check.Name); exists {
			go domainCheckTimer(check, fn)
		}
	}
}

func domainCheckTimer(check cfg.Check, fn CheckDomainFunc) {
	time.Sleep(3 * time.Second)
	runDomainCheck(check, fn)

	ticker := time.NewTicker(time.Duration(check.CheckInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		runDomainCheck(check, fn)
	}
}

func runDomainCheck(check cfg.Check, fn CheckDomainFunc) {
	c := cfg.GetConfig()

	for svcName, svc := range c.Services {
		for _, member := range c.Members {
			if member.Membership.Level >= svc.Configuration.LevelRequired &&
				member.Service.Active == 1 && !member.Override {

				domains := make(map[string]struct{})

				for _, assignments := range member.ServiceAssignments {
					for _, assignment := range assignments {
						if assignment == svcName {
							for _, provider := range svc.Providers {
								for _, rpcURL := range provider.RpcUrls {
									if domain := parseUrlForDomain(rpcURL); domain != "" {
										domains[domain] = struct{}{}
									}
								}
							}
						}
					}
				}

				for domain := range domains {
					go domainCheckWrapper(check, fn, domain, svc, member)
					time.Sleep(30 * time.Millisecond)
				}
			}
		}
	}
}

func domainCheckWrapper(check cfg.Check, fn CheckDomainFunc, domain string, service cfg.Service, member cfg.Member) {
	done := make(chan struct{})
	timer := time.NewTimer(time.Duration(check.Timeout) * time.Second)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Log(log.Error, "Check %s for member %s crashed: %v", check.Name, member.Details.Name, r)
				UpdateDomainResultLocal(check, domain, service, member, false, "Check crashed", nil, false)
				UpdateDomainResultLocal(check, domain, service, member, false, "Check crashed", nil, true)
			}
			close(done)
		}()
		fn(check, domain, service, member)
	}()

	select {
	case <-done:
	case <-timer.C:
		UpdateDomainResultLocal(check, domain, service, member, false, "Check timed out", nil, false)
		UpdateDomainResultLocal(check, domain, service, member, false, "Check timed out", nil, true)
	}
}

func UpdateDomainResultLocal(check cfg.Check, domain string, service cfg.Service, member cfg.Member,
	status bool, errorMsg string, dataMap map[string]interface{}, isIPv6 bool) {

	dat.UpdateLocalDomainResult(check, member, service, domain, status, errorMsg, dataMap, isIPv6)

	found, officialStatus := dat.GetOfficialDomainStatus(check.Name, member.Details.Name, domain, isIPv6)
	if !found {
		natsCommon.ProposeCheckStatus("domain", check.Name, member.Details.Name, domain, "", status, errorMsg, dataMap, isIPv6)
		return
	}
	if officialStatus != status {
		natsCommon.ProposeCheckStatus("domain", check.Name, member.Details.Name, domain, "", status, errorMsg, dataMap, isIPv6)
	}
}

// ------------------------------------------------------------------
// ENDPOINT checks (unchanged except parseUrlForDomain call)
// ------------------------------------------------------------------

func getEndpointCheck(name string) (CheckEndpointFunc, bool) {
	CheckRegistry.Mu.RLock()
	defer CheckRegistry.Mu.RUnlock()
	fn, ok := CheckRegistry.Endpoint[name]
	return fn, ok
}

func initEndpointCheck() {
	c := cfg.GetConfig()
	var endpointChecks []cfg.Check
	for _, ch := range c.Local.Checks {
		if ch.CheckType == "endpoint" && ch.Enabled == 1 {
			endpointChecks = append(endpointChecks, ch)
		}
	}
	for _, check := range endpointChecks {
		if fn, exists := getEndpointCheck(check.Name); exists {
			go endpointCheckTimer(check, fn)
		}
	}
}

func endpointCheckTimer(check cfg.Check, fn CheckEndpointFunc) {
	time.Sleep(4 * time.Second)
	runEndpointCheck(check, fn)

	ticker := time.NewTicker(time.Duration(check.CheckInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		runEndpointCheck(check, fn)
	}
}

func runEndpointCheck(check cfg.Check, fn CheckEndpointFunc) {
	c := cfg.GetConfig()
	for svcName, svc := range c.Services {
		for _, member := range c.Members {
			if member.Membership.Level >= svc.Configuration.LevelRequired &&
				member.Service.Active == 1 && !member.Override {

				for _, assignments := range member.ServiceAssignments {
					for _, assignment := range assignments {
						if assignment == svcName {
							for _, provider := range svc.Providers {
								for _, rpcURL := range provider.RpcUrls {
									go endpointCheckWrapper(check, fn, rpcURL, svc, member)
									time.Sleep(30 * time.Millisecond)
								}
							}
						}
					}
				}
			}
		}
	}
}

func endpointCheckWrapper(check cfg.Check, fn CheckEndpointFunc, endpoint string, service cfg.Service, member cfg.Member) {
	done := make(chan struct{})
	timer := time.NewTimer(time.Duration(check.Timeout) * time.Second)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Log(log.Error, "Check %s for member %s crashed: %v", check.Name, member.Details.Name, r)
				UpdateEndpointResultLocal(check, member, service, endpoint, false, "Check crashed", nil, false)
				UpdateEndpointResultLocal(check, member, service, endpoint, false, "Check crashed", nil, true)
			}
			close(done)
		}()
		fn(check, endpoint, service, member)
	}()

	select {
	case <-done:
	case <-timer.C:
		UpdateEndpointResultLocal(check, member, service, endpoint, false, "Check timed out", nil, false)
		UpdateEndpointResultLocal(check, member, service, endpoint, false, "Check timed out", nil, true)
	}
}

func UpdateEndpointResultLocal(
	check cfg.Check,
	member cfg.Member,
	service cfg.Service,
	endpoint string,
	status bool,
	errorMsg string,
	dataMap map[string]interface{},
	isIPv6 bool,
) {
	domain := parseUrlForDomain(endpoint)
	dat.UpdateLocalEndpointResult(check, member, service, domain, endpoint, status, errorMsg, dataMap, isIPv6)

	found, officialStatus := dat.GetOfficialEndpointStatus(check.Name, member.Details.Name, domain, endpoint, isIPv6)
	if !found {
		natsCommon.ProposeCheckStatus("endpoint", check.Name, member.Details.Name, domain, endpoint, status, errorMsg, dataMap, isIPv6)
		return
	}
	if officialStatus != status {
		natsCommon.ProposeCheckStatus("endpoint", check.Name, member.Details.Name, domain, endpoint, status, errorMsg, dataMap, isIPv6)
	}
}

// ------------------------------------------------------------------
// Helpers
// ------------------------------------------------------------------

// parseUrlForDomain normalises any RPC / WSS URL into a **host‑only** string
// without a port, always lower‑cased.  It tolerates missing scheme and
// returns an empty string on irreversible errors.
func parseUrlForDomain(raw string) string {
	if raw == "" {
		return ""
	}

	// Ensure we can parse: add dummy scheme if absent.
	testStr := raw
	if !strings.Contains(raw, "://") {
		testStr = "https://" + raw // scheme never impacts Hostname()
	}

	u, err := url.Parse(testStr)
	if err != nil {
		return ""
	}

	host := u.Hostname() // strips port automatically
	return strings.ToLower(host)
}

// parseUrlForDomain is also required by some Monitor modules outside this file.
var _ = max.ParseUrl
