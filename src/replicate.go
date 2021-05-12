package src

import (
	"database/sql"
	"log"

	"github.com/georgysavva/scany/dbscan"
	"github.com/georgysavva/scany/sqlscan"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

func ReadHighestBlock(db *sql.DB) (*Block, error) {
	rows, err := db.Query("select * from blocks where block_height = (select max(block_height) from blocks)")
	if err != nil {
		return nil, errors.Wrap(err, "failed to query latest block")
	}

	var block Block
	err = dbscan.ScanOne(&block, rows)
	if err != nil {
		if dbscan.NotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to scan block")
	}

	return &block, nil
}

func ReadMaxReplicatedBlockHeight(db *sql.DB) (string, error) {
	row := db.QueryRow("SELECT IFNULL(MAX(block_height), 0) FROM replication_meta")
	var height string
	err := row.Scan(&height)
	if err != nil {
		return "", errors.Wrap(err, "failed to query latest block")
	}
	return height, nil
}

func WriteReplicatedBlockHeight(db *sql.DB, block_height string) error {
	_, err := db.Exec("REPLACE INTO replication_meta VALUES (?)", block_height)
	return errors.Wrap(err, "failed to save replicated block height")
}

func Replicate(pgConn *sql.DB, sdbConn *sql.DB, baseHeight string, limit int) (string, error) {
	loader := NewLoader(sdbConn)

	rows, err := pgConn.Query("select * from blocks where block_height >= $1 order by block_height asc limit $2", baseHeight, limit)
	if err != nil {
		return "", errors.Wrap(err, "failed to read blocks")
	}

	scanner := sqlscan.NewRowScanner(rows)
	blockHashes := make([]string, 0)
	var maxBlockHeight string
	dst := &Block{}
	for rows.Next() {
		err := scanner.Scan(dst)
		if err != nil {
			return "", errors.Wrap(err, "failed to scan into &Block{}")
		}
		err = loader.WriteRow("blocks", dst)
		if err != nil {
			return "", errors.Wrap(err, "failed to write row to loader")
		}
		blockHashes = append(blockHashes, dst.Key())
		maxBlockHeight = dst.BlockHeight
		MetricReplicatedRows.Inc()
	}

	if len(blockHashes) == 0 {
		return "", nil
	}

	simpleReplicate := func(collectKeys bool, table string, dst Model, query string, args ...interface{}) ([]string, error) {
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
		return "", errors.Wrap(err, "failed to replicate transactions")
	}

	receiptIDs, err := simpleReplicate(true, "receipts", &Receipt{}, "select * from receipts where included_in_block_hash = ANY($1)", pq.Array(blockHashes))
	if err != nil {
		return "", errors.Wrap(err, "failed to replicate receipts")
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

	simpleReplicateParallel("chunks", &Chunk{}, "select * from chunks where included_in_block_hash = ANY($1)", pq.Array(blockHashes))

	simpleReplicateParallel("execution_outcomes", &ExecutionOutcome{}, "select * from execution_outcomes where executed_in_block_hash = ANY($1)", pq.Array(blockHashes))

	simpleReplicateParallel("access_keys", &AccessKey{}, "select * from access_keys where last_update_block_height >= $1", baseHeight)

	simpleReplicateParallel("accounts", &Account{}, "select * from accounts where last_update_block_height >= $1", baseHeight)

	simpleReplicateParallel("transaction_actions", &TransactionAction{}, "select * from transaction_actions where transaction_hash = ANY($1)", pq.Array(transactionHashes))

	simpleReplicateParallel("action_receipts", &ActionReceipt{}, "select * from action_receipts where receipt_id = ANY($1)", pq.Array(receiptIDs))

	simpleReplicateParallel("data_receipts", &DataReceipt{}, "select * from data_receipts where receipt_id = ANY($1)", pq.Array(receiptIDs))

	simpleReplicateParallel("action_receipt_actions", &ActionReceiptAction{}, "select * from action_receipt_actions where receipt_id = ANY($1)", pq.Array(receiptIDs))

	simpleReplicateParallel("action_receipt_input_data", &ActionReceiptInputData{}, "select * from action_receipt_input_data where input_to_receipt_id = ANY($1)", pq.Array(receiptIDs))

	simpleReplicateParallel("action_receipt_output_data", &ActionReceiptOutputData{}, "select * from action_receipt_output_data where output_from_receipt_id = ANY($1)", pq.Array(receiptIDs))

	simpleReplicateParallel("execution_outcome_receipts", &ExecutionOutcomeReceipt{}, "select * from execution_outcome_receipts where executed_receipt_id = ANY($1) or produced_receipt_id = ANY($1)", pq.Array(receiptIDs))

	var lastError error
	for i := 0; i < numParallel; i++ {
		err = <-results
		if err != nil {
			log.Printf("error while loading into SingleStore: %+v", err)
			lastError = err
		}
	}
	if lastError != nil {
		return "", lastError
	}

	err = loader.Close()
	if err != nil {
		return "", errors.Wrap(err, "failed to finalize the load")
	}

	return maxBlockHeight, nil
}
