package monitor

import (
	cfg "ibp-geodns/src/common/config"
	dat "ibp-geodns/src/common/data"
	log "ibp-geodns/src/common/logging"
	max "ibp-geodns/src/common/maxmind"
	sig "ibp-geodns/src/common/signal"
	"sync"
	"time"
)

// Create an instance of ChecksRegistry
var CheckRegistry = struct {
	Site     map[string]CheckSiteFunc
	Domain   map[string]CheckDomainFunc
	Endpoint map[string]CheckEndpointFunc
	Mu       sync.RWMutex
}{
	Site:     make(map[string]CheckSiteFunc, 0),
	Domain:   make(map[string]CheckDomainFunc, 0),
	Endpoint: make(map[string]CheckEndpointFunc, 0),
}

func startChecks() {
	go InitSiteCheck()
	go InitDomainCheck()
	go InitEndpointCheck()
}

type CheckSiteFunc func(check cfg.Check, member cfg.Member)

func RegisterSiteCheck(name string, checkFunc CheckSiteFunc) {
	CheckRegistry.Site[name] = checkFunc
	log.Log(log.Info, "Registered site check '%s'", name)
}

func GetSiteCheck(name string) (CheckSiteFunc, bool) {
	for checkName, checkFunc := range CheckRegistry.Site {
		if checkName == name {
			return checkFunc, true
		}
	}
	return nil, false
}

func InitSiteCheck() {
	c := cfg.GetConfig()

	for _, check := range c.Local.Checks {
		if check.CheckType == "site" {
			checkFunc, exists := GetSiteCheck(check.Name)
			if exists {
				log.Log(log.Info, "Site check %s detected. Timer Activated.", check.Name)
				go SiteCheckTimer(check, checkFunc)
			}
		}
	}
}

func SiteCheckTimer(check cfg.Check, checkFunc CheckSiteFunc) {
	time.Sleep(2 * time.Second)
	RunSiteCheck(check, checkFunc)

	ticker := time.NewTicker(time.Duration(check.CheckInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		RunSiteCheck(check, checkFunc)
	}
}

func RunSiteCheck(check cfg.Check, checkFunc CheckSiteFunc) {
	c := cfg.GetConfig()

	for _, member := range c.Members {
		if (member.Service.Active == 1) && (!member.Override) {
			go SiteCheckWrapper(check, checkFunc, member)
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func SiteCheckWrapper(check cfg.Check, checkFunc CheckSiteFunc, member cfg.Member) {
	done := make(chan struct{})
	timer := time.NewTimer(time.Duration(check.Timeout) * time.Second)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Log(log.Error, "Check %s for member %s crashed: %v", check.Name, member.Details.Name, r)
				go UpdateSiteResultLocal(check, member, false, "Check crashed", map[string]interface{}{})
				close(done)
			}
		}()
		checkFunc(check, member)
		close(done)
	}()

	select {
	case <-done:
	case <-timer.C:
		go UpdateSiteResultLocal(check, member, false, "Check timed out", map[string]interface{}{})
	}
}

func UpdateSiteResultLocal(check cfg.Check, member cfg.Member, status bool, errorMsg string, dataMap map[string]interface{}) {
	go dat.UpdateLocalSiteResult(check, member, status, errorMsg, dataMap)

	// Check difference
	exists, OfficialStatus := dat.GetOfficialSiteStatus(check.Name, member.Details.Name)

	if !exists {
		go sig.ProposeCheckStatus("site", check.Name, member.Details.Name, "", "", status, errorMsg, dataMap)
	} else {
		if OfficialStatus != status {
			go sig.ProposeCheckStatus("site", check.Name, member.Details.Name, "", "", status, errorMsg, dataMap)
		}
	}
}

/*
 *
 *  Functions for Domain checks
 *
 */

type CheckDomainFunc func(check cfg.Check, domain string, service cfg.Service, member cfg.Member)

func RegisterDomainCheck(name string, checkFunc CheckDomainFunc) {
	CheckRegistry.Domain[name] = checkFunc
	log.Log(log.Info, "Registered domain check '%s'", name)
}

func GetDomainCheck(name string) (CheckDomainFunc, bool) {
	for checkName, checkFunc := range CheckRegistry.Domain {
		if checkName == name {
			return checkFunc, true
		}
	}
	return nil, false
}

func InitDomainCheck() {
	c := cfg.GetConfig()

	for _, check := range c.Local.Checks {
		if check.CheckType == "domain" {
			checkFunc, exists := GetDomainCheck(check.Name)
			if exists {
				log.Log(log.Info, "Domain check %s detected. Timer Activated.", check.Name)
				go DomainCheckTimer(check, checkFunc)
			}
		}
	}
}

func DomainCheckTimer(check cfg.Check, checkFunc CheckDomainFunc) {
	time.Sleep(3 * time.Second)
	RunDomainCheck(check, checkFunc)

	ticker := time.NewTicker(time.Duration(check.CheckInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		RunDomainCheck(check, checkFunc)
	}
}

func RunDomainCheck(check cfg.Check, checkFunc CheckDomainFunc) {
	c := cfg.GetConfig()

	for serviceName, service := range c.Services {
		for _, member := range c.Members {
			if (member.Membership.Level >= service.Configuration.LevelRequired) &&
				(member.Service.Active == 1) &&
				(!member.Override) {

				uniqueDomains := make(map[string]struct{})

				for _, assignments := range member.ServiceAssignments {
					for _, assignment := range assignments {
						if assignment == serviceName {
							for _, provider := range service.Providers {
								for _, url := range provider.RpcUrls {
									parsed := max.ParseUrl(url)
									uniqueDomains[parsed.Domain] = struct{}{}
								}
							}
						}
					}
				}

				for domain := range uniqueDomains {
					go DomainCheckWrapper(check, checkFunc, domain, service, member)
					time.Sleep(10 * time.Millisecond)
				}
			}
		}
	}
}

func DomainCheckWrapper(check cfg.Check, checkFunc CheckDomainFunc, domain string, service cfg.Service, member cfg.Member) {
	done := make(chan struct{})
	timer := time.NewTimer(time.Duration(check.Timeout) * time.Second)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Log(log.Error, "Check %s for member %s crashed: %v", check.Name, member.Details.Name, r)
				go UpdateDomainResultLocal(check, domain, service, member, false, "Check crashed", map[string]interface{}{})
				close(done)
			}
		}()
		checkFunc(check, domain, service, member)
		close(done)
	}()

	select {
	case <-done:
	case <-timer.C:
		go UpdateDomainResultLocal(check, domain, service, member, false, "Check timed out", map[string]interface{}{})
	}
}

func UpdateDomainResultLocal(check cfg.Check, domain string, service cfg.Service, member cfg.Member, status bool, errorMsg string, dataMap map[string]interface{}) {
	go dat.UpdateLocalDomainResult(check, member, service, domain, status, errorMsg, dataMap)

	// Check difference
	exists, OfficialStatus := dat.GetOfficialDomainStatus(check.Name, member.Details.Name, domain)

	if !exists {
		go sig.ProposeCheckStatus("domain", check.Name, member.Details.Name, domain, "", status, errorMsg, dataMap)
	} else {
		if OfficialStatus != status {
			go sig.ProposeCheckStatus("domain", check.Name, member.Details.Name, domain, "", status, errorMsg, dataMap)
		}
	}
}

type CheckEndpointFunc func(check cfg.Check, endpoint string, service cfg.Service, member cfg.Member)

func RegisterEndpointCheck(name string, checkFunc CheckEndpointFunc) {
	CheckRegistry.Endpoint[name] = checkFunc
	log.Log(log.Info, "Registered endpoint check '%s'", name)
}

func GetEndpointCheck(name string) (CheckEndpointFunc, bool) {
	checkFunc, exists := CheckRegistry.Endpoint[name]
	return checkFunc, exists
}

func InitEndpointCheck() {
	c := cfg.GetConfig()

	for _, check := range c.Local.Checks {
		if check.CheckType == "endpoint" {
			checkFunc, exists := GetEndpointCheck(check.Name)
			if exists {
				log.Log(log.Info, "Endpoint check %s detected. Timer Activated.", check.Name)
				go EndpointCheckTimer(check, checkFunc)
			}
		}
	}
}

func EndpointCheckTimer(check cfg.Check, checkFunc CheckEndpointFunc) {
	time.Sleep(4 * time.Second)
	RunEndpointCheck(check, checkFunc)

	ticker := time.NewTicker(time.Duration(check.CheckInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		RunEndpointCheck(check, checkFunc)
	}
}

func RunEndpointCheck(check cfg.Check, checkFunc CheckEndpointFunc) {
	c := cfg.GetConfig()

	for serviceName, service := range c.Services {
		for _, member := range c.Members {
			if (member.Membership.Level >= service.Configuration.LevelRequired) &&
				(member.Service.Active == 1) &&
				(!member.Override) {

				for _, assignments := range member.ServiceAssignments {
					for _, assignment := range assignments {
						if assignment == serviceName {
							for _, provider := range service.Providers {
								for _, url := range provider.RpcUrls {
									go EndpointCheckWrapper(check, checkFunc, url, service, member)
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

func EndpointCheckWrapper(check cfg.Check, checkFunc CheckEndpointFunc, endpoint string, service cfg.Service, member cfg.Member) {
	done := make(chan struct{})
	timer := time.NewTimer(time.Duration(check.Timeout) * time.Second)
	u := max.ParseUrl(endpoint)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Log(log.Error, "Check %s for member %s crashed: %v", check.Name, member.Details.Name, r)
				go UpdateEndpointResultLocal(check, member, service, u.Domain, endpoint, false, "Check crashed", map[string]interface{}{})
				close(done)
			}
		}()
		checkFunc(check, endpoint, service, member)
		close(done)
	}()

	select {
	case <-done:
	case <-timer.C:
		go UpdateEndpointResultLocal(check, member, service, u.Domain, endpoint, false, "Check timed out", map[string]interface{}{})
	}
}

func UpdateEndpointResultLocal(check cfg.Check, member cfg.Member, service cfg.Service, domain string, endpoint string, status bool, errorMsg string, dataMap map[string]interface{}) {
	u := max.ParseUrl(endpoint)

	go dat.UpdateLocalEndpointResult(check, member, service, u.Domain, endpoint, status, errorMsg, dataMap)

	// Check difference
	exists, OfficialStatus := dat.GetOfficialEndpointStatus(check.Name, member.Details.Name, u.Domain, endpoint)

	if !exists {
		go sig.ProposeCheckStatus("endpoint", check.Name, member.Details.Name, u.Domain, endpoint, status, errorMsg, dataMap)
	} else {
		if OfficialStatus != status {
			go sig.ProposeCheckStatus("endpoint", check.Name, member.Details.Name, u.Domain, endpoint, status, errorMsg, dataMap)
		}
	}
}
