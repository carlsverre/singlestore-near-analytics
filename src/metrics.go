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
)

func ServeMetrics(config MetricsConfig) {
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
	if err != nil {
		log.Fatalf("failed to start metrics server: %s", err)
	}
}
