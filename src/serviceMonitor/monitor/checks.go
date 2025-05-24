package monitor

import (
	"time"

	cfg "ibp-geodns/src/common/config"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
	nats "ibp-geodns/src/common/nats"
	"sync"
)

// All checks are stored here
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

// CheckSiteFunc signature
type CheckSiteFunc func(check cfg.Check, member cfg.Member)

// CheckDomainFunc signature
type CheckDomainFunc func(check cfg.Check, domain string, service cfg.Service, member cfg.Member)

// CheckEndpointFunc signature
type CheckEndpointFunc func(check cfg.Check, endpoint string, service cfg.Service, member cfg.Member)

func startChecks() {
	go initSiteCheck()
	go initDomainCheck()
	go initEndpointCheck()
}

func RegisterSiteCheck(name string, checkFunc CheckSiteFunc) {
	CheckRegistry.Mu.Lock()
	defer CheckRegistry.Mu.Unlock()
	CheckRegistry.Site[name] = checkFunc
	log.Log(log.Info, "Registered site check '%s'", name)
}

func RegisterDomainCheck(name string, checkFunc CheckDomainFunc) {
	CheckRegistry.Mu.Lock()
	defer CheckRegistry.Mu.Unlock()
	CheckRegistry.Domain[name] = checkFunc
	log.Log(log.Info, "Registered domain check '%s'", name)
}

func RegisterEndpointCheck(name string, checkFunc CheckEndpointFunc) {
	CheckRegistry.Mu.Lock()
	defer CheckRegistry.Mu.Unlock()
	CheckRegistry.Endpoint[name] = checkFunc
	log.Log(log.Info, "Registered endpoint check '%s'", name)
}

func getSiteCheck(name string) (CheckSiteFunc, bool) {
	CheckRegistry.Mu.RLock()
	defer CheckRegistry.Mu.RUnlock()
	fn, ok := CheckRegistry.Site[name]
	return fn, ok
}

func getDomainCheck(name string) (CheckDomainFunc, bool) {
	CheckRegistry.Mu.RLock()
	defer CheckRegistry.Mu.RUnlock()
	fn, ok := CheckRegistry.Domain[name]
	return fn, ok
}

func getEndpointCheck(name string) (CheckEndpointFunc, bool) {
	CheckRegistry.Mu.RLock()
	defer CheckRegistry.Mu.RUnlock()
	fn, ok := CheckRegistry.Endpoint[name]
	return fn, ok
}

// ------------------- Site checks

func initSiteCheck() {
	c := cfg.GetConfig()
	for _, check := range c.Local.Checks {
		if check.CheckType == "site" && check.Enabled == 1 {
			fn, exists := getSiteCheck(check.Name)
			if exists {
				go siteCheckTimer(check, fn)
			}
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
							UpdateSiteResultLocal(ch, m, false, "Check crashed", nil)
						}
						close(done)
					}()
					fn(ch, m)
				}()

				select {
				case <-done:
				case <-timer.C:
					UpdateSiteResultLocal(ch, m, false, "Check timed out", nil)
				}
			}(check, member)
		}
	}
}

func UpdateSiteResultLocal(check cfg.Check, member cfg.Member, status bool, errorMsg string, dataMap map[string]interface{}) {
	updateLocalSiteResults(check, member, status, errorMsg, dataMap)
}

// We update local site results, then we propose a new status if there's a discrepancy with official
func updateLocalSiteResults(check cfg.Check, member cfg.Member, status bool, errorText string, data map[string]interface{}) {
	exists, offStatus := getOfficialSiteStatus(check.Name, member.Details.Name)
	if !exists {
		nats.ProposeCheckStatus("site", check.Name, member.Details.Name, "", "", status, errorText, data)
		return
	}
	if offStatus != status {
		nats.ProposeCheckStatus("site", check.Name, member.Details.Name, "", "", status, errorText, data)
	}
}

// --------------- Domain checks

func initDomainCheck() {
	c := cfg.GetConfig()
	for _, check := range c.Local.Checks {
		if check.CheckType == "domain" && check.Enabled == 1 {
			fn, exists := getDomainCheck(check.Name)
			if exists {
				go domainCheckTimer(check, fn)
			}
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
									parsed := max.ParseUrl(url)
									domainsSet[parsed.Domain] = struct{}{}
								}
							}
						}
					}
				}
				for dom := range domainsSet {
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
				UpdateDomainResultLocal(check, domain, service, member, false, "Check crashed", nil)
			}
			close(done)
		}()
		fn(check, domain, service, member)
	}()

	select {
	case <-done:
	case <-timer.C:
		UpdateDomainResultLocal(check, domain, service, member, false, "Check timed out", nil)
	}
}

func UpdateDomainResultLocal(check cfg.Check, domain string, service cfg.Service, member cfg.Member, status bool, errorText string, data map[string]interface{}) {
	exists, offStatus := getOfficialDomainStatus(check.Name, member.Details.Name, domain)
	if !exists {
		nats.ProposeCheckStatus("domain", check.Name, member.Details.Name, domain, "", status, errorText, data)
		return
	}
	if offStatus != status {
		nats.ProposeCheckStatus("domain", check.Name, member.Details.Name, domain, "", status, errorText, data)
	}
}

// --------------- Endpoint checks

func initEndpointCheck() {
	c := cfg.GetConfig()
	for _, check := range c.Local.Checks {
		if check.CheckType == "endpoint" && check.Enabled == 1 {
			fn, exists := getEndpointCheck(check.Name)
			if exists {
				go endpointCheckTimer(check, fn)
			}
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
				UpdateEndpointResultLocal(check, member, service, endpoint, false, "Check crashed", nil)
			}
			close(done)
		}()
		fn(check, endpoint, service, member)
	}()

	select {
	case <-done:
	case <-timer.C:
		UpdateEndpointResultLocal(check, member, service, endpoint, false, "Check timed out", nil)
	}
}

func UpdateEndpointResultLocal(check cfg.Check, member cfg.Member, service cfg.Service, endpoint string, status bool, errorText string, data map[string]interface{}) {
	u := max.ParseUrl(endpoint)

	exists, offStatus := getOfficialEndpointStatus(check.Name, member.Details.Name, u.Domain, endpoint)
	if !exists {
		nats.ProposeCheckStatus("endpoint", check.Name, member.Details.Name, u.Domain, endpoint, status, errorText, data)
		return
	}
	if offStatus != status {
		nats.ProposeCheckStatus("endpoint", check.Name, member.Details.Name, u.Domain, endpoint, status, errorText, data)
	}
}
