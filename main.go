package main

import (
	"flag"
	"log"
	"time"
)

var configPath = flag.String("config", "config.yaml", "path to the config file")
var startHeight = flag.Int64("start-height", -1, "start replicating at this block height")
var batchSize = flag.Int("batch-size", 100, "maximum number of blocks to replicate per batch")
var pollInterval = flag.Duration("poll-interval", time.Millisecond*500, "time to sleep between polling postgres for more blocks")

func main() {
	flag.Parse()
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	if configPath == nil || *configPath == "" {
		log.Fatal("--config is required")
	}

	config, err := ParseConfig(*configPath)
	if err != nil {
		log.Fatalf("unable to load config file: %s; error: %+v", *configPath, err)
	}

	go ServeMetrics(config.Metrics)

	pgConn, err := ConnectPostgres(config.Postgres)
	if err != nil {
		log.Fatalf("unable to connect to postgres: %+v", err)
	}
	defer pgConn.Close()

	sdbConn, err := ConnectSingleStore(config.SingleStore)
	if err != nil {
		log.Fatalf("unable to connect to singlestore: %+v", err)
	}
	defer sdbConn.Close()

	log.Printf("starting replication from postgres (%s:%d) to singlestore (%s:%d)",
		config.Postgres.Host, config.Postgres.Port,
		config.SingleStore.Host, config.SingleStore.Port)
	log.Printf("metrics available at http://localhost:%d/metrics", config.Metrics.Port)

	if *startHeight == -1 {
		highestBlockSingleStore, err := ReadHighestBlock(sdbConn)
		if err != nil {
			log.Fatalf("unable to read highest block from singlestore: %+v", err)
		}
		if highestBlockSingleStore == nil {
			log.Fatal("refusing to start from the first block; specify --start-height=0 to override")
		}
		startHeight = &highestBlockSingleStore.BlockHeight
		// start replicating at the next block
		(*startHeight)++
	}

	highestBlockPostgres, err := ReadHighestBlock(pgConn)
	if err != nil {
		log.Fatalf("unable to read highest block from postgres: %+v", err)
	}

	height := *startHeight
	limit := *batchSize
	for {
		err = Replicate(pgConn, sdbConn, height, limit)
		if err != nil {
			log.Fatalf("replication failed: %+v", err)
		}

		highestBlockSingleStore, err := ReadHighestBlock(sdbConn)
		if err != nil {
			log.Fatalf("unable to read highest block from singlestore: %+v", err)
		}
		height = highestBlockSingleStore.BlockHeight + 1

		metricBlockHeight.Set(float64(height))

		// only sleep if we have "caught up"
		if highestBlockSingleStore.BlockHeight > highestBlockPostgres.BlockHeight {
			time.Sleep(*pollInterval)
		} else {
			log.Printf("replicated up to block %d, target %d", highestBlockSingleStore.BlockHeight, highestBlockPostgres.BlockHeight)
		}
	}
}
