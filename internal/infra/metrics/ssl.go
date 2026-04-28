package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	registerOnce sync.Once

	sslRenewScheduledTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "ventopanel",
		Subsystem: "ssl",
		Name:      "renew_scheduled_total",
		Help:      "Total number of SSL renew tasks scheduled.",
	})

	sslRenewSuccessTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "ventopanel",
		Subsystem: "ssl",
		Name:      "renew_success_total",
		Help:      "Total number of successful SSL renew operations.",
	})

	sslRenewFailedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "ventopanel",
		Subsystem: "ssl",
		Name:      "renew_failed_total",
		Help:      "Total number of failed SSL renew operations.",
	})

	sslLastBatchServerCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "ventopanel",
		Subsystem: "ssl",
		Name:      "last_batch_server_count",
		Help:      "Number of servers in the last scheduled SSL renew batch.",
	})
)

func Register() {
	registerOnce.Do(func() {
		prometheus.MustRegister(
			sslRenewScheduledTotal,
			sslRenewSuccessTotal,
			sslRenewFailedTotal,
			sslLastBatchServerCount,
			aclDeniedTotal,
		)
	})
}

func IncSSLRenewScheduled() {
	Register()
	sslRenewScheduledTotal.Inc()
}

func IncSSLRenewSuccess() {
	Register()
	sslRenewSuccessTotal.Inc()
}

func IncSSLRenewFailed() {
	Register()
	sslRenewFailedTotal.Inc()
}

func SetLastBatchServerCount(count int) {
	Register()
	sslLastBatchServerCount.Set(float64(count))
}
