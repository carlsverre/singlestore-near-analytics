package main

import (
	"flag"
	"fmt"
	"log"
	"math/big"
	"time"

	"f0a.org/singlestore-near-analytics/src"
)

var configPath = flag.String("config", "config.yaml", "path to the config file")
var startHeight = flag.String("start-height", "-1", "start replicating at this block height")
var batchSize = flag.Int("batch-size", 100, "maximum number of blocks to replicate per batch")
var pollInterval = flag.Duration("poll-interval", time.Millisecond*500, "time to sleep between polling postgres for more blocks")

func parseBigInt(i string) *big.Int {
	out, ok := (&big.Int{}).SetString(i, 10)
	if !ok {
		panic(fmt.Sprintf("failed to parse big.Int: %s", i))
	}
	return out
}

func incrementHeight(height string) string {
	return (&big.Int{}).Add(parseBigInt(height), big.NewInt(1)).String()
}

func compareHeights(left, right string) int {
	return parseBigInt(left).Cmp(parseBigInt(right))
}

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

	height := *startHeight

	if height == "-1" {
		var err error
		height, err = src.ReadMaxReplicatedBlockHeight(sdbConn)
		if err != nil {
			log.Fatalf("unable to read highest block from singlestore: %+v", err)
		}
		if height == "0" {
			log.Fatal("refusing to start from the first block; specify `--start-height 0` to override")
		}
		// start replicating at the next block
		height = incrementHeight(height)
	}

	highestBlockPostgres, err := src.ReadHighestBlock(pgConn)
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

		src.MetricBatchReplicationTime.Observe(time.Now().Sub(start).Seconds())

		if replicatedHeight != "" {
			err = src.WriteReplicatedBlockHeight(sdbConn, replicatedHeight)
			if err != nil {
				log.Fatalf("replication failed: %+v", err)
			}

			height = incrementHeight(replicatedHeight)
		}

		// only sleep if we have "caught up"
		if compareHeights(height, highestBlockPostgres.BlockHeight) >= 0 {
			time.Sleep(interval)
		} else {
			log.Printf("catching up to height %s, currently at height %s", highestBlockPostgres.BlockHeight, height)
		}
	}
}
