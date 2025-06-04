package monitor

import (
	"sync"
	"time"

	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
	natsCommon "ibp-geodns/src/common/nats"
)

// Registry of checks
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

// Function types for different checks
type (
	CheckSiteFunc     func(check cfg.Check, member cfg.Member)
	CheckDomainFunc   func(check cfg.Check, domain string, service cfg.Service, member cfg.Member)
	CheckEndpointFunc func(check cfg.Check, endpoint string, service cfg.Service, member cfg.Member)
)

// RegisterSiteCheck ...
func RegisterSiteCheck(name string, checkFunc CheckSiteFunc) {
	CheckRegistry.Mu.Lock()
	defer CheckRegistry.Mu.Unlock()
	CheckRegistry.Site[name] = checkFunc
	log.Log(log.Info, "Registered site check '%s'", name)
}

// RegisterDomainCheck ...
func RegisterDomainCheck(name string, checkFunc CheckDomainFunc) {
	CheckRegistry.Mu.Lock()
	defer CheckRegistry.Mu.Unlock()
	CheckRegistry.Domain[name] = checkFunc
	log.Log(log.Info, "Registered domain check '%s'", name)
}

// RegisterEndpointCheck ...
func RegisterEndpointCheck(name string, checkFunc CheckEndpointFunc) {
	CheckRegistry.Mu.Lock()
	defer CheckRegistry.Mu.Unlock()
	CheckRegistry.Endpoint[name] = checkFunc
}

// startChecks is invoked by monitor.Init() to kick off all checks in parallel.
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
		fn, exists := getSiteCheck(check.Name)
		if exists {
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

				go func() {
					defer func() {
						if r := recover(); r != nil {
							log.Log(log.Error, "Check %s for member %s crashed: %v", ch.Name, m.Details.Name, r)
							UpdateSiteResultLocal(ch, m, false, "Check crashed", nil, false)
						}
						close(done)
					}()
					fn(ch, m)
				}()

				select {
				case <-done:
				case <-timer.C:
					UpdateSiteResultLocal(ch, m, false, "Check timed out", nil, false)
				}
			}(check, member)
		}
	}
}

// UpdateSiteResultLocal includes isIPv6
func UpdateSiteResultLocal(check cfg.Check, member cfg.Member, status bool, errorMsg string, dataMap map[string]interface{}, isIPv6 bool) {
	// 1) Store in data.Local
	dat.UpdateLocalSiteResult(check, member, status, errorMsg, dataMap, isIPv6)

	// 2) Compare with official
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
		fn, exists := getDomainCheck(check.Name)
		if exists {
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
				domainsSet := make(map[string]struct{})
				for _, assignments := range member.ServiceAssignments {
					for _, assignment := range assignments {
						if assignment == svcName {
							for _, provider := range svc.Providers {
								for _, url := range provider.RpcUrls {
									parsed := parseUrlForDomain(url)
									if parsed != "" {
										domainsSet[parsed] = struct{}{}
									}
								}
							}
						}
					}
				}
				for dom := range domainsSet {
					// We'll do separate calls for IPv4 vs IPv6 if present
					go domainCheckWrapper(check, fn, dom, svc, member)
					time.Sleep(10 * time.Millisecond)
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
			}
			close(done)
		}()
		fn(check, domain, service, member)
	}()

	select {
	case <-done:
	case <-timer.C:
		UpdateDomainResultLocal(check, domain, service, member, false, "Check timed out", nil, false)
	}
}

// We introduce a helper for parsing domain out of an URL without pulling in maxmind parse code here
func parseUrlForDomain(raw string) string {
	// naive approach
	// or we could do something simpler since maxmind is not imported here
	// but let's do a quick parse
	// remove protocol
	// e.g. wss://mydomain.com/path => mydomain.com
	// strip path
	// return domain

	// we can do something minimal
	// user specifically said we do not rely on advanced logic here, it's domain only
	// The code below is stable enough

	start := 0
	if idx := indexOf(raw, "://"); idx != -1 {
		start = idx + 3
	}
	rest := raw[start:]
	if slash := indexOf(rest, "/"); slash != -1 {
		rest = rest[:slash]
	}
	return rest
}

func indexOf(str, sep string) int {
	returnIndex := -1
	for i := 0; i+len(sep) <= len(str); i++ {
		if str[i:i+len(sep)] == sep {
			returnIndex = i
			break
		}
	}
	return returnIndex
}

// UpdateDomainResultLocal adds isIPv6 param
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
// ENDPOINT checks
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
		fn, exists := getEndpointCheck(check.Name)
		if exists {
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
								for _, url := range provider.RpcUrls {
									go endpointCheckWrapper(check, fn, url, svc, member)
									time.Sleep(10 * time.Millisecond)
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
			}
			close(done)
		}()
		fn(check, endpoint, service, member)
	}()

	select {
	case <-done:
	case <-timer.C:
		UpdateEndpointResultLocal(check, member, service, endpoint, false, "Check timed out", nil, false)
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
