package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTP
	RequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total HTTP requests",
		},
		[]string{"route", "method", "status"},
	)

	// İşlemler
	TransactionsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "transactions_total",
			Help: "Total successful transactions",
		},
		[]string{"type"}, // credit|debit|transfer
	)
	TransactionsFailed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "transactions_failed_total",
			Help: "Total failed transactions",
		},
	)

	// Worker kuyruğu
	WorkerQueueDepth = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "worker_queue_depth",
			Help: "Current worker queue depth",
		},
	)
)

// /metrics endpoint'i için handler
var Handler = promhttp.Handler

func Init() {
	prometheus.MustRegister(RequestsTotal)
	prometheus.MustRegister(TransactionsTotal)
	prometheus.MustRegister(TransactionsFailed)
	prometheus.MustRegister(WorkerQueueDepth)
}
