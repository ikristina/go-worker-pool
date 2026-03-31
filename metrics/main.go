package metrics

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	JobsTotal     *prometheus.CounterVec
	JobDuration   prometheus.Histogram
	WorkersActive prometheus.Gauge
}

func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		JobsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "jobs_total",
				Help: "Total number of jobs processed, by status.",
			},
			[]string{"status"},
		),
		JobDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "job_duration_seconds",
			Help:    "Time spent executing a job.",
			Buckets: []float64{0.1, 0.25, 0.5, 1, 2.5, 5},
		}),
		WorkersActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "workers_active",
			Help: "Number of workers currently executing a job.",
		}),
	}
	reg.MustRegister(m.JobsTotal)
	reg.MustRegister(m.JobDuration)
	reg.MustRegister(m.WorkersActive)
	reg.MustRegister(prometheus.NewGoCollector())
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	return m
}
