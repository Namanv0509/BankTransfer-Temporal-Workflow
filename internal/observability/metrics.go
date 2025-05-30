package observability

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	WorkflowDuration *prometheus.HistogramVec
	ActivityFailures *prometheus.CounterVec
}

func NewMetrics() *Metrics {
	m := &Metrics{
		WorkflowDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "workflow_execution_duration_seconds",
				Help: "Duration of workflow execution",
			},
			[]string{"workflow", "status"},
		),
		ActivityFailures: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "activity_failures_total",
				Help: "Total number of activity failures",
			},
			[]string{"activity"},
		),
	}
	prometheus.MustRegister(m.WorkflowDuration, m.ActivityFailures)
	return m
}
