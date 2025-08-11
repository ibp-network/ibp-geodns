package api

import (
	"net/http"
	"sort"
	"strings"

	cfg "ibp-geodns/src/common/config"
)

// ServiceInfo represents the enhanced service information
type ServiceInfo struct {
	Name          string                `json:"name"`
	DisplayName   string                `json:"display_name"`
	ServiceType   string                `json:"service_type"`
	NetworkName   string                `json:"network_name"`
	RelayNetwork  string                `json:"relay_network"` // NEW FIELD
	NetworkType   string                `json:"network_type"`  // NEW FIELD
	WebsiteURL    string                `json:"website_url"`
	LogoURL       string                `json:"logo_url"`
	Description   string                `json:"description"`
	Active        bool                  `json:"active"`
	LevelRequired int                   `json:"level_required"`
	Resources     cfg.Resources         `json:"resources"`
	Providers     []ServiceProviderInfo `json:"providers"`
	MemberCount   int                   `json:"member_count"`
	TotalCost     float64               `json:"total_monthly_cost"`
}

// ServiceProviderInfo represents provider information
type ServiceProviderInfo struct {
	Name    string   `json:"name"`
	RpcUrls []string `json:"rpc_urls"`
}

// ServiceHierarchy represents the hierarchical organization of services
type ServiceHierarchy struct {
	RelayChains []RelayChainInfo `json:"relay_chains"`
	Orphans     []ServiceInfo    `json:"orphans"` // Services without a relay network
}

// RelayChainInfo represents a relay chain with its associated chains
type RelayChainInfo struct {
	Relay     ServiceInfo   `json:"relay"`
	System    []ServiceInfo `json:"system_chains"`
	Community []ServiceInfo `json:"community_chains"`
}

// handleServices returns all services with enhanced information
func handleServices(w http.ResponseWriter, r *http.Request) {
	c := cfg.GetConfig()
	serviceName := r.URL.Query().Get("name")

	// If specific service requested
	if serviceName != "" {
		service, exists := c.Services[serviceName]
		if !exists {
			writeError(w, http.StatusNotFound, "Service not found")
			return
		}

		serviceInfo := buildServiceInfo(serviceName, service, c)
		writeJSON(w, http.StatusOK, serviceInfo)
		return
	}

	// Check if hierarchy view is requested
	if r.URL.Query().Get("hierarchy") == "true" {
		hierarchy := buildServiceHierarchy(c)
		writeJSON(w, http.StatusOK, hierarchy)
		return
	}

	// Return all services (flat list)
	services := []ServiceInfo{}
	for name, service := range c.Services {
		services = append(services, buildServiceInfo(name, service, c))
	}

	// Sort by name
	sort.Slice(services, func(i, j int) bool {
		return strings.ToLower(services[i].Name) < strings.ToLower(services[j].Name)
	})

	result := map[string]interface{}{
		"services": services,
		"total":    len(services),
	}

	writeJSON(w, http.StatusOK, result)
}

// buildServiceHierarchy organizes services into a hierarchical structure
func buildServiceHierarchy(config cfg.Config) ServiceHierarchy {
	hierarchy := ServiceHierarchy{
		RelayChains: []RelayChainInfo{},
		Orphans:     []ServiceInfo{},
	}

	// First pass: identify relay chains
	relayServices := make(map[string]ServiceInfo)
	systemServices := make(map[string][]ServiceInfo)
	communityServices := make(map[string][]ServiceInfo)

	for name, service := range config.Services {
		serviceInfo := buildServiceInfo(name, service, config)

		// Check if it's a relay chain (RelayNetwork is empty and NetworkType is "Relay")
		if service.Configuration.RelayNetwork == "" && service.Configuration.NetworkType == "Relay" {
			relayServices[name] = serviceInfo
		} else if service.Configuration.RelayNetwork != "" {
			// It's a parachain - categorize by type
			relayName := service.Configuration.RelayNetwork

			switch service.Configuration.NetworkType {
			case "System":
				if systemServices[relayName] == nil {
					systemServices[relayName] = []ServiceInfo{}
				}
				systemServices[relayName] = append(systemServices[relayName], serviceInfo)
			case "Community":
				if communityServices[relayName] == nil {
					communityServices[relayName] = []ServiceInfo{}
				}
				communityServices[relayName] = append(communityServices[relayName], serviceInfo)
			default:
				// Unknown type - add to orphans
				hierarchy.Orphans = append(hierarchy.Orphans, serviceInfo)
			}
		} else {
			// No relay network and not a relay - add to orphans
			if service.Configuration.NetworkType != "Relay" {
				hierarchy.Orphans = append(hierarchy.Orphans, serviceInfo)
			}
		}
	}

	// Build relay chain hierarchy
	for relayName, relayInfo := range relayServices {
		relayChain := RelayChainInfo{
			Relay:     relayInfo,
			System:    systemServices[relayName],
			Community: communityServices[relayName],
		}

		// Sort system chains by name
		if relayChain.System != nil {
			sort.Slice(relayChain.System, func(i, j int) bool {
				return strings.ToLower(relayChain.System[i].DisplayName) < strings.ToLower(relayChain.System[j].DisplayName)
			})
		}

		// Sort community chains by name
		if relayChain.Community != nil {
			sort.Slice(relayChain.Community, func(i, j int) bool {
				return strings.ToLower(relayChain.Community[i].DisplayName) < strings.ToLower(relayChain.Community[j].DisplayName)
			})
		}

		hierarchy.RelayChains = append(hierarchy.RelayChains, relayChain)
	}

	// Sort relay chains by name
	sort.Slice(hierarchy.RelayChains, func(i, j int) bool {
		return strings.ToLower(hierarchy.RelayChains[i].Relay.DisplayName) < strings.ToLower(hierarchy.RelayChains[j].Relay.DisplayName)
	})

	// Sort orphans
	sort.Slice(hierarchy.Orphans, func(i, j int) bool {
		return strings.ToLower(hierarchy.Orphans[i].DisplayName) < strings.ToLower(hierarchy.Orphans[j].DisplayName)
	})

	return hierarchy
}

// handleServicesSummary returns a summary of all services
func handleServicesSummary(w http.ResponseWriter, r *http.Request) {
	c := cfg.GetConfig()

	// Count services by type and network type
	serviceTypes := make(map[string]int)
	networkTypes := make(map[string]int)
	relayChainCount := 0
	activeCount := 0
	totalResources := cfg.Resources{}

	for _, service := range c.Services {
		serviceTypes[service.Configuration.ServiceType]++
		networkTypes[service.Configuration.NetworkType]++

		if service.Configuration.NetworkType == "Relay" && service.Configuration.RelayNetwork == "" {
			relayChainCount++
		}

		if service.Configuration.Active == 1 {
			activeCount++
		}

		// Sum resources
		totalResources.Nodes += service.Resources.Nodes
		totalResources.Cores += service.Resources.Cores * float64(service.Resources.Nodes)
		totalResources.Memory += service.Resources.Memory * float64(service.Resources.Nodes)
		totalResources.Disk += service.Resources.Disk * float64(service.Resources.Nodes)
		totalResources.Bandwidth += service.Resources.Bandwidth * float64(service.Resources.Nodes)
	}

	summary := map[string]interface{}{
		"total_services":  len(c.Services),
		"active_services": activeCount,
		"relay_chains":    relayChainCount,
		"service_types":   serviceTypes,
		"network_types":   networkTypes,
		"total_resources": totalResources,
	}

	writeJSON(w, http.StatusOK, summary)
}

func buildServiceInfo(name string, service cfg.Service, config cfg.Config) ServiceInfo {
	// Count members assigned to this service
	memberCount := 0
	for _, member := range config.Members {
		if member.Service.Active == 1 && !member.Override {
			for _, assignments := range member.ServiceAssignments {
				for _, svcName := range assignments {
					if svcName == name {
						memberCount++
						break
					}
				}
			}
		}
	}

	// Build providers list
	providers := []ServiceProviderInfo{}
	for provName, provider := range service.Providers {
		providers = append(providers, ServiceProviderInfo{
			Name:    provName,
			RpcUrls: provider.RpcUrls,
		})
	}

	// Sort providers by name
	sort.Slice(providers, func(i, j int) bool {
		return strings.ToLower(providers[i].Name) < strings.ToLower(providers[j].Name)
	})

	// Calculate total cost (simplified - you may want to use the billing calculation)
	totalCost := 0.0
	// This is a placeholder - integrate with your billing calculation

	return ServiceInfo{
		Name:          name,
		DisplayName:   service.Configuration.DisplayName,
		ServiceType:   service.Configuration.ServiceType,
		NetworkName:   service.Configuration.NetworkName,
		RelayNetwork:  service.Configuration.RelayNetwork,
		NetworkType:   service.Configuration.NetworkType,
		WebsiteURL:    service.Configuration.WebsiteURL,
		LogoURL:       service.Configuration.LogoURL,
		Description:   service.Configuration.Description,
		Active:        service.Configuration.Active == 1,
		LevelRequired: service.Configuration.LevelRequired,
		Resources:     service.Resources,
		Providers:     providers,
		MemberCount:   memberCount,
		TotalCost:     totalCost,
	}
}
