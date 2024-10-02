package powerdns

import (
	"net/http"
	"log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Define metrics
var (
	serviceStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ibp_geodns_service_up",
			Help: "Service status for each member in the GeoDNS network",
		},
		[]string{"memberName", "serviceId"},
	)
	memberPingStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ibp_geodns_member_ping",
			Help: "Member ping status in the GeoDNS network",
		},
		[]string{"memberName"},
	)
	memberOverrideStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ibp_geodns_member_override",
			Help: "Member override status in the GeoDNS network",
		},
		[]string{"memberName"},
	)
)

// Register metrics
func init() {
	prometheus.Unregister(prometheus.NewGoCollector())
	log.Println("Registering prometheus metrics")
	prometheus.MustRegister(memberPingStatus)
	prometheus.MustRegister(memberOverrideStatus)
	prometheus.MustRegister(serviceStatus)
}

// MetricsHandler serves the /metrics endpoint
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// UpdateMetrics parses API results and updates Prometheus metrics
func UpdateMetrics(data []DNS) {
	// log.Printf("Updating metrics with data: %+v\n", data)
	// Clear previous metrics
	serviceStatus.Reset()

	// Iterate over the DNS configs and update the metrics
	for _, dns := range data {
		for memberName, member := range dns.Members {

			status := 0.0
			override := member.Override
			if override {
				status = 1.0
			}
			memberOverrideStatus.WithLabelValues(memberName).Set(status)

			for serviceId, result := range member.Results {
				// Set 1 if success is true, otherwise set 0
				status := 0.0
				if result.Success {
					status = 1.0
				}
				if serviceId == "ping" {
					log.Printf("Setting ping status for %s: %w\n", memberName, result)
					memberPingStatus.WithLabelValues(memberName).Set(status)
				} else {
					// Update the gauge metric for this member and service
					serviceStatus.WithLabelValues(memberName, serviceId).Set(status)
				}
			}
		}
	}
}
