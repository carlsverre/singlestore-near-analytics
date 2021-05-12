# SingleStore NEAR Analytics

This project provides a simple replication tool for continually replicating data
from the public NEAR analytics dataset.

You can learn more about the NEAR analytics dataset here: https://github.com/near/near-indexer-for-explorer

## Setup

1. Clone this repo
2. Copy config.yaml.example to config.yaml and update with real connection details
3. Copy initialize/config.env.example to initialize/config.env and update with real connection details
4. apply schema.sql to your SingleStore cluster

## Initial Load

For the initial load, use the python code contained in ./initialize which is more efficient at doing the initial bulk copy than the go code.

1. Install dependencies
    ```bash
    sudo apt install lz4 postgresql mariadb-client python3 python3-pip
    pip3 install -r initialize/requirements.txt
    ```
2. Run the initial copy (this will take some time)
    ```bash
    cd initialize
    source config.env
    python3 run.py
    ```

## Continuous Replication

1. Run the replication tool

    ```bash
    go build
    ./singlestore-near-analytics
    ```

## Prometheus Metrics

The replication tool exports prometheus metrics at localhost:9000/metrics (by default, override in config). To consume them locally you can spin up prometheus in docker like so:

```bash
docker run -it --net host -v $PWD/prometheus.yaml:/etc/prometheus/prometheus.yml prom/prometheus
```

Then visit localhost:9090 to query the metrics.
