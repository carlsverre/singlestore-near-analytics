package main

import (
	"flag"
	"log"
	"time"
)

var configPath = flag.String("config", "config.yaml", "path to the config file")

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

	for {
		err = Replicate(pgConn, sdbConn)
		if err != nil {
			log.Fatalf("replication failed: %+v", err)
		}
		time.Sleep(time.Second)
	}
}
