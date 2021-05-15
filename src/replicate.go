package src

import (
	"database/sql"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/georgysavva/scany/sqlscan"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

func readMaxBlockHeightFromTable(db *sql.DB, tableName string) (*big.Int, error) {
	row := db.QueryRow(fmt.Sprintf("SELECT coalesce(MAX(block_height), 0) FROM %s", tableName))
	var height string
	err := row.Scan(&height)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query latest block")
	}
	return ParseBigInt(height), nil
}

func ReadMaxReplicatedBlockHeight(db *sql.DB) (*big.Int, error) {
	return readMaxBlockHeightFromTable(db, "replication_meta")
}

func ReadMaxBlockHeight(db *sql.DB) (*big.Int, error) {
	return readMaxBlockHeightFromTable(db, "blocks")
}

func MonitorBlockHeights(pgConn *sql.DB, sdbConn *sql.DB, pollInterval time.Duration) {
	var (
		pgHeight  *big.Int
		sdbHeight *big.Int
		err       error

		pgGauge  = MetricBlockHeight.WithLabelValues("postgres")
		sdbGauge = MetricBlockHeight.WithLabelValues("singlestore")
	)

	for {
		pgHeight, err = ReadMaxBlockHeight(pgConn)
		if err != nil {
			log.Printf("failed to read from postgres: %+v", err)
		}

		sdbHeight, err = ReadMaxReplicatedBlockHeight(sdbConn)
		if err != nil {
			log.Printf("failed to read from singlestore: %+v", err)
		}

		// this will stop working once height > 2^63-1
		// on the plus side, that's going to be awhile
		pgGauge.Set(float64(pgHeight.Int64()))
		sdbGauge.Set(float64(sdbHeight.Int64()))

		// track lag for convenience
		lag := (&big.Int{}).Sub(pgHeight, sdbHeight).Int64()
		MetricBlockLag.Set(float64(lag))

		time.Sleep(pollInterval)
	}
}

func WriteReplicatedBlockHeight(db *sql.DB, block_height *big.Int) error {
	_, err := db.Exec("REPLACE INTO replication_meta VALUES (?)", block_height.String())
	return errors.Wrap(err, "failed to save replicated block height")
}

func Replicate(pgConn *sql.DB, sdbConn *sql.DB, baseHeight *big.Int, limit int) (*big.Int, error) {
	rowCount := pgConn.QueryRow("select count(*) from blocks where block_height >= $1", baseHeight.String())
	var blockCount int64
	err := rowCount.Scan(&blockCount)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read count from row")
	}
	if blockCount == 0 {
		return nil, nil
	}

	loader := NewLoader(sdbConn)

	err = loader.Touch("blocks")
	if err != nil {
		return nil, err
	}

	rows, err := pgConn.Query("select * from blocks where block_height >= $1 order by block_height asc limit $2", baseHeight.String(), limit)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read blocks")
	}

	scanner := sqlscan.NewRowScanner(rows)
	blockHashes := make([]string, 0)
	var maxBlockHeight string
	dst := &Block{}
	for rows.Next() {
		err := scanner.Scan(dst)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan into &Block{}")
		}
		err = loader.WriteRow("blocks", dst)
		if err != nil {
			return nil, errors.Wrap(err, "failed to write row to loader")
		}
		blockHashes = append(blockHashes, dst.Key())
		maxBlockHeight = dst.BlockHeight
		MetricReplicatedRows.Inc()
		MetricReplicatedBlocks.Inc()
	}

	MetricBatchSize.Set(float64(len(blockHashes)))

	simpleReplicate := func(collectKeys bool, table string, dst Model, query string, args ...interface{}) ([]string, error) {
		err := loader.Touch(table)
		if err != nil {
			return nil, err
		}

		rows, err := pgConn.Query(query, args...)
		if err != nil {
			return nil, err
		}

		scanner := sqlscan.NewRowScanner(rows)
		keys := make([]string, 0)
		for rows.Next() {
			err := scanner.Scan(dst)
			if err != nil {
				return nil, err
			}
			err = loader.WriteRow(table, dst)
			if err != nil {
				return nil, err
			}
			if collectKeys {
				keys = append(keys, dst.Key())
			}
			MetricReplicatedRows.Inc()
		}
		return keys, nil
	}

	transactionHashes, err := simpleReplicate(true, "transactions", &Transaction{}, "select * from transactions where included_in_block_hash = ANY($1)", pq.Array(blockHashes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to replicate transactions")
	}

	receiptIDs, err := simpleReplicate(true, "receipts", &Receipt{}, "select * from receipts where included_in_block_hash = ANY($1)", pq.Array(blockHashes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to replicate receipts")
	}

	numParallel := 0
	results := make(chan error)

	simpleReplicateParallel := func(table string, dst Model, query string, args ...interface{}) {
		numParallel++
		go func() {
			_, err = simpleReplicate(false, table, dst, query, args...)
			if err != nil {
				results <- errors.Wrapf(err, "failed to replicate %s", table)
			} else {
				results <- nil
			}
		}()
	}

	simpleReplicateParallel("access_keys", &AccessKey{}, "select * from access_keys where last_update_block_height >= $1", baseHeight.String())

	simpleReplicateParallel("account_changes", &AccountChange{}, "select * from account_changes where changed_in_block_hash = ANY($1)", pq.Array(blockHashes))

	simpleReplicateParallel("accounts", &Account{}, "select * from accounts where last_update_block_height >= $1", baseHeight.String())

	simpleReplicateParallel("action_receipt_actions", &ActionReceiptAction{}, "select * from action_receipt_actions where receipt_id = ANY($1)", pq.Array(receiptIDs))

	simpleReplicateParallel("action_receipt_input_data", &ActionReceiptInputData{}, "select * from action_receipt_input_data where input_to_receipt_id = ANY($1)", pq.Array(receiptIDs))

	simpleReplicateParallel("action_receipt_output_data", &ActionReceiptOutputData{}, "select * from action_receipt_output_data where output_from_receipt_id = ANY($1)", pq.Array(receiptIDs))

	simpleReplicateParallel("action_receipts", &ActionReceipt{}, "select * from action_receipts where receipt_id = ANY($1)", pq.Array(receiptIDs))

	simpleReplicateParallel("chunks", &Chunk{}, "select * from chunks where included_in_block_hash = ANY($1)", pq.Array(blockHashes))

	simpleReplicateParallel("data_receipts", &DataReceipt{}, "select * from data_receipts where receipt_id = ANY($1)", pq.Array(receiptIDs))

	simpleReplicateParallel("execution_outcome_receipts", &ExecutionOutcomeReceipt{}, "select * from execution_outcome_receipts where executed_receipt_id = ANY($1) or produced_receipt_id = ANY($1)", pq.Array(receiptIDs))

	simpleReplicateParallel("execution_outcomes", &ExecutionOutcome{}, "select * from execution_outcomes where executed_in_block_hash = ANY($1)", pq.Array(blockHashes))

	simpleReplicateParallel("transaction_actions", &TransactionAction{}, "select * from transaction_actions where transaction_hash = ANY($1)", pq.Array(transactionHashes))

	var lastError error
	for i := 0; i < numParallel; i++ {
		err = <-results
		if err != nil {
			log.Printf("error while loading into SingleStore: %+v", err)
			lastError = err
		}
	}
	if lastError != nil {
		return nil, lastError
	}

	err = loader.Close()
	if err != nil {
		return nil, errors.Wrap(err, "failed to finalize the load")
	}

	if untouched := loader.UntouchedTables(); len(untouched) > 0 {
		return nil, errors.Errorf("the following tables are not being replicated to: %v", untouched)
	}

	return ParseBigInt(maxBlockHeight), nil
}
