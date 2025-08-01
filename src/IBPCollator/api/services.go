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

	// Return all services
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

// handleServicesSummary returns a summary of all services
func handleServicesSummary(w http.ResponseWriter, r *http.Request) {
	c := cfg.GetConfig()

	// Count services by type
	serviceTypes := make(map[string]int)
	activeCount := 0
	totalResources := cfg.Resources{}

	for _, service := range c.Services {
		serviceTypes[service.Configuration.ServiceType]++
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
		"service_types":   serviceTypes,
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
