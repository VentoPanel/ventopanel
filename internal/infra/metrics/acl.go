package metrics

import "github.com/prometheus/client_golang/prometheus"

var aclDeniedTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "ventopanel",
		Subsystem: "acl",
		Name:      "denied_total",
		Help:      "Total number of ACL denied decisions.",
	},
	[]string{"resource_type", "reason"},
)

func IncACLDenied(resourceType, reason string) {
	Register()
	aclDeniedTotal.WithLabelValues(resourceType, reason).Inc()
}
