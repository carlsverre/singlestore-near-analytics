package src

import (
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	MetricReplicatedRows = promauto.NewCounter(prometheus.CounterOpts{
		Name: "singlestore_replicated_rows",
		Help: "The total number of rows replicated to SingleStore",
	})

	MetricReplicatedBlocks = promauto.NewCounter(prometheus.CounterOpts{
		Name: "singlestore_replicated_blocks",
		Help: "The total number of blocks replicated to SingleStore",
	})

	MetricBatchReplicationTime = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "singlestore_replication_duration_seconds",
		Help:    "Measures the time it takes to replicate a batch to SingleStore",
		Buckets: []float64{0.0125, 0.025, 0.05, 0.1, 0.2, 0.4, 0.8, 1.6, 3.2, 6.4, 12.8, 24.6},
	})
)

func ServeMetrics(config MetricsConfig) {
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
	if err != nil {
		log.Fatalf("failed to start metrics server: %s", err)
	}
}
