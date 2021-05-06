# SingleStore NEAR Analytics

This project provides a simple replication tool for continually replicating data
from the public NEAR analytics dataset.

You can learn more about the NEAR analytics dataset here: https://github.com/near/near-indexer-for-explorer

## Setup

1. Clone this repo
2. Copy config.yaml.example to config.yaml and modify
3. `go build`
4. Run the replication tool
    ```
    ./singlestore-near-analytics
    ```