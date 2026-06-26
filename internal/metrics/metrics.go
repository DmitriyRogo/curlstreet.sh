package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "curlstreet_http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path", "status"},
	)

	CacheOps = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "curlstreet_cache_operations_total",
			Help: "Cache operations by result (hit or miss).",
		},
		[]string{"result"},
	)

	ProviderRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "curlstreet_provider_request_duration_seconds",
			Help:    "Upstream provider request latency in seconds.",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"provider", "endpoint"},
	)

	ProviderErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "curlstreet_provider_errors_total",
			Help: "Provider errors by provider and error type.",
		},
		[]string{"provider", "error"},
	)
)

func Register() {
	prometheus.MustRegister(
		HTTPRequestDuration,
		CacheOps,
		ProviderRequestDuration,
		ProviderErrors,
	)
}
