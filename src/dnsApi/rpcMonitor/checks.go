package rpcMonitor

import (
	"ibp-geodns/src/common/config"
	"ibp-geodns/src/common/data"
	l "ibp-geodns/src/common/logging"
	g "ibp-geodns/src/common/maxmind"
	nComm "ibp-geodns/src/common/nodeComm"
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

/*
 *
 *  Functions for Site checks
 *
 */

type CheckSiteFunc func(check config.Check, member config.Member)

func RegisterSiteCheck(name string, checkFunc CheckSiteFunc) {
	CheckRegistry.Site[name] = checkFunc
	l.Log(l.Debug, "Registered site check '%s'", name)
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
	c := config.GetConfig()

	for _, check := range c.System.Checks {
		if check.CheckType == "site" {
			checkFunc, exists := GetSiteCheck(check.Name)
			if exists {
				l.Log(l.Debug, "Site check %s detected. Timer Activated.", check.Name)
				go SiteCheckTimer(check, checkFunc)
			}
		}
	}
}

func SiteCheckTimer(check config.Check, checkFunc CheckSiteFunc) {
	time.Sleep(2 * time.Second)
	RunSiteCheck(check, checkFunc)

	ticker := time.NewTicker(time.Duration(check.CheckInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		RunSiteCheck(check, checkFunc)
	}
}

func RunSiteCheck(check config.Check, checkFunc CheckSiteFunc) {
	c := config.GetConfig()

	for _, member := range c.Members {
		if (member.Service.Active == 1) && (!member.Override) {
			go SiteCheckWrapper(check, checkFunc, member)
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func SiteCheckWrapper(check config.Check, checkFunc CheckSiteFunc, member config.Member) {
	done := make(chan struct{})
	timer := time.NewTimer(time.Duration(check.Timeout) * time.Second)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				l.Log(l.Debug, "Check %s for member %s crashed: %v", check.Name, member.Details.Name, r)
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

func UpdateSiteResultLocal(check config.Check, member config.Member, status bool, errorMsg string, dataMap map[string]interface{}) {
	go data.UpdateLocalSiteResult(check, member, status, errorMsg, dataMap)

	// Check difference
	exists, OfficialStatus := data.GetOfficialSiteStatus(check.Name, member.Details.Name)

	if !exists {
		go nComm.ProposeCheckStatus("site", check.Name, member.Details.Name, "", "", status, errorMsg, dataMap)
	} else {
		if OfficialStatus != status {
			go nComm.ProposeCheckStatus("site", check.Name, member.Details.Name, "", "", status, errorMsg, dataMap)
		}
	}
}

/*
 *
 *  Functions for Domain checks
 *
 */

type CheckDomainFunc func(check config.Check, domain string, service config.Service, member config.Member)

func RegisterDomainCheck(name string, checkFunc CheckDomainFunc) {
	CheckRegistry.Domain[name] = checkFunc
	l.Log(l.Debug, "Registered domain check '%s'", name)
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
	c := config.GetConfig()

	for _, check := range c.System.Checks {
		if check.CheckType == "domain" {
			checkFunc, exists := GetDomainCheck(check.Name)
			if exists {
				l.Log(l.Debug, "Domain check %s detected. Timer Activated.", check.Name)
				go DomainCheckTimer(check, checkFunc)
			}
		}
	}
}

func DomainCheckTimer(check config.Check, checkFunc CheckDomainFunc) {
	time.Sleep(3 * time.Second)
	RunDomainCheck(check, checkFunc)

	ticker := time.NewTicker(time.Duration(check.CheckInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		RunDomainCheck(check, checkFunc)
	}
}

func RunDomainCheck(check config.Check, checkFunc CheckDomainFunc) {
	c := config.GetConfig()

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
									parsed := g.ParseUrl(url)
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

func DomainCheckWrapper(check config.Check, checkFunc CheckDomainFunc, domain string, service config.Service, member config.Member) {
	done := make(chan struct{})
	timer := time.NewTimer(time.Duration(check.Timeout) * time.Second)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				l.Log(l.Debug, "Check %s for member %s crashed: %v", check.Name, member.Details.Name, r)
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

func UpdateDomainResultLocal(check config.Check, domain string, service config.Service, member config.Member, status bool, errorMsg string, dataMap map[string]interface{}) {
	go data.UpdateLocalDomainResult(check, member, service, domain, status, errorMsg, dataMap)

	// Check difference
	exists, OfficialStatus := data.GetOfficialDomainStatus(check.Name, member.Details.Name, domain)

	if !exists {
		go nComm.ProposeCheckStatus("domain", check.Name, member.Details.Name, domain, "", status, errorMsg, dataMap)
	} else {
		if OfficialStatus != status {
			go nComm.ProposeCheckStatus("domain", check.Name, member.Details.Name, domain, "", status, errorMsg, dataMap)
		}
	}
}

/*
 *
 *  Functions for Endpoint checks
 *
 */

type CheckEndpointFunc func(check config.Check, endpoint string, service config.Service, member config.Member)

func RegisterEndpointCheck(name string, checkFunc CheckEndpointFunc) {
	CheckRegistry.Endpoint[name] = checkFunc
	l.Log(l.Debug, "Registered endpoint check '%s'", name)
}

func GetEndpointCheck(name string) (CheckEndpointFunc, bool) {
	checkFunc, exists := CheckRegistry.Endpoint[name]
	return checkFunc, exists
}

func InitEndpointCheck() {
	c := config.GetConfig()

	for _, check := range c.System.Checks {
		if check.CheckType == "endpoint" {
			checkFunc, exists := GetEndpointCheck(check.Name)
			if exists {
				l.Log(l.Debug, "Endpoint check %s detected. Timer Activated.", check.Name)
				go EndpointCheckTimer(check, checkFunc)
			}
		}
	}
}

func EndpointCheckTimer(check config.Check, checkFunc CheckEndpointFunc) {
	time.Sleep(4 * time.Second)
	RunEndpointCheck(check, checkFunc)

	ticker := time.NewTicker(time.Duration(check.CheckInterval) * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		RunEndpointCheck(check, checkFunc)
	}
}

func RunEndpointCheck(check config.Check, checkFunc CheckEndpointFunc) {
	c := config.GetConfig()

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

func EndpointCheckWrapper(check config.Check, checkFunc CheckEndpointFunc, endpoint string, service config.Service, member config.Member) {
	done := make(chan struct{})
	timer := time.NewTimer(time.Duration(check.Timeout) * time.Second)
	u := g.ParseUrl(endpoint)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				l.Log(l.Debug, "Check %s for member %s crashed: %v", check.Name, member.Details.Name, r)
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

func UpdateEndpointResultLocal(check config.Check, member config.Member, service config.Service, domain string, endpoint string, status bool, errorMsg string, dataMap map[string]interface{}) {
	u := g.ParseUrl(endpoint)

	go data.UpdateLocalEndpointResult(check, member, service, u.Domain, endpoint, status, errorMsg, dataMap)

	// Check difference
	exists, OfficialStatus := data.GetOfficialEndpointStatus(check.Name, member.Details.Name, u.Domain, endpoint)

	if !exists {
		go nComm.ProposeCheckStatus("endpoint", check.Name, member.Details.Name, u.Domain, endpoint, status, errorMsg, dataMap)
	} else {
		if OfficialStatus != status {
			go nComm.ProposeCheckStatus("endpoint", check.Name, member.Details.Name, u.Domain, endpoint, status, errorMsg, dataMap)
		}
	}
}
