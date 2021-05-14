package main

import (
	"flag"
	"log"
	"math/big"
	"time"

	"f0a.org/singlestore-near-analytics/src"
)

var configPath = flag.String("config", "config.yaml", "path to the config file")
var startHeight = flag.String("start-height", "-1", "start replicating at this block height")
var batchSize = flag.Int("batch-size", 100, "maximum number of blocks to replicate per batch")
var pollInterval = flag.Duration("poll-interval", time.Millisecond*500, "time to sleep between polling postgres for more blocks")

func main() {
	flag.Parse()
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	if configPath == nil || *configPath == "" {
		log.Fatal("--config is required")
	}

	config, err := src.ParseConfig(*configPath)
	if err != nil {
		log.Fatalf("unable to load config file: %s; error: %+v", *configPath, err)
	}

	go src.ServeMetrics(config.Metrics)

	pgConn, err := src.ConnectPostgres(config.Postgres)
	if err != nil {
		log.Fatalf("unable to connect to postgres: %+v", err)
	}
	defer pgConn.Close()

	sdbConn, err := src.ConnectSingleStore(config.SingleStore)
	if err != nil {
		log.Fatalf("unable to connect to singlestore: %+v", err)
	}
	defer sdbConn.Close()

	log.Printf("starting replication from postgres (%s:%d) to singlestore (%s:%d)",
		config.Postgres.Host, config.Postgres.Port,
		config.SingleStore.Host, config.SingleStore.Port)
	log.Printf("metrics available at http://localhost:%d/metrics", config.Metrics.Port)

	go src.MonitorBlockHeights(pgConn, sdbConn, time.Second)

	height := src.ParseBigInt(*startHeight)

	if height.Cmp(big.NewInt(-1)) == 0 {
		var err error
		height, err = src.ReadMaxReplicatedBlockHeight(sdbConn)
		if err != nil {
			log.Fatalf("unable to read highest block from singlestore: %+v", err)
		}
		if height.Cmp(big.NewInt(0)) == 0 {
			log.Fatal("refusing to start from the first block; specify `--start-height 0` to override")
		}
		// start replicating at the next block
		height.Add(height, big.NewInt(1))
	}

	pgInitialMaxBlockHeight, err := src.ReadMaxBlockHeight(pgConn)
	if err != nil {
		log.Fatalf("unable to read highest block from postgres: %+v", err)
	}

	log.Printf("starting replication at block height = %s", height)

	limit := *batchSize
	interval := *pollInterval
	for {
		start := time.Now()

		replicatedHeight, err := src.Replicate(pgConn, sdbConn, height, limit)
		if err != nil {
			log.Fatalf("replication failed: %+v", err)
		}

		replicationDuration := time.Now().Sub(start)

		if replicatedHeight != nil {
			// only record the replication time metric if we actually replicated something
			src.MetricBatchReplicationTime.Observe(replicationDuration.Seconds())

			err = src.WriteReplicatedBlockHeight(sdbConn, replicatedHeight)
			if err != nil {
				log.Fatalf("replication failed: %+v", err)
			}

			height = replicatedHeight.Add(replicatedHeight, big.NewInt(1))
		}

		// only sleep if we have "caught up"
		if height.Cmp(pgInitialMaxBlockHeight) >= 0 {
			time.Sleep(interval - replicationDuration)
		} else {
			log.Printf("catching up to height %s, currently at height %s", pgInitialMaxBlockHeight, height)
		}
	}
}
