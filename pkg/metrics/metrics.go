// SPDX-License-Identifier: MIT

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const (
	metricControllerLabel = "controller"
	metricControllerValue = "garm_operator"
	metricNamespace       = "garm_operator"
	garmAPI               = "api_requests"
	garmInfo              = "info"
)

var (
	GarmJwtExpiresAt = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: metricNamespace,
		Subsystem: garmInfo,
		Name:      "jwt_expires_at",
		Help:      "Expiry date of obtained garm-server JWT",
		ConstLabels: prometheus.Labels{
			metricControllerLabel: metricControllerValue,
		},
	})

	TotalGarmCalls = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricNamespace,
			Subsystem: garmAPI,
			Name:      "total",
			Help:      "Total number of GARM API calls",
			ConstLabels: prometheus.Labels{
				metricControllerLabel: metricControllerValue,
			},
		}, []string{"method"})

	GarmCallErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricNamespace,
			Subsystem: garmAPI,
			Name:      "errors_total",
			Help:      "Number of GARM API calls that failed",
			ConstLabels: prometheus.Labels{
				metricControllerLabel: metricControllerValue,
			},
		}, []string{"method"})
)

func init() {
	metrics.Registry.MustRegister(GarmJwtExpiresAt)
	metrics.Registry.MustRegister(TotalGarmCalls)
	metrics.Registry.MustRegister(GarmCallErrors)
}
