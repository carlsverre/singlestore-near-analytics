package main

import (
	"database/sql"
	"log"

	"github.com/georgysavva/scany/sqlscan"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

func Replicate(pgConn *sql.DB, sdbConn *sql.DB) error {
	var baseHeight int64
	err := sdbConn.QueryRow("select IFNULL(max(block_height), 0) from blocks").Scan(&baseHeight)
	if err != nil {
		return errors.Wrap(err, "failed to query max block height")
	}

	if baseHeight == 0 {
		return errors.New("refusing to replicate from beginning of time")
	}

	blockHeight.Set(float64(baseHeight))

	loader := NewLoader(sdbConn, []string{
		"blocks",
		"chunks",
		"transactions",
		"receipts",
		"execution_outcomes",
		"access_keys",
		"accounts",
		"account_changes",
		"transaction_actions",
		"action_receipts",
		"data_receipts",
		"action_receipt_actions",
		"action_receipt_input_data",
		"action_receipt_output_data",
		"execution_outcome_receipts",
	})

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
			err = loader.WriteRow(table, dst.Record())
			if err != nil {
				return nil, err
			}
			if collectKeys {
				keys = append(keys, dst.Key())
			}
			replicatedRows.Inc()
		}
		return keys, nil
	}

	blockHashes, err := simpleReplicate(true, "blocks", &Block{}, "select * from blocks where block_height > $1 order by block_height asc", baseHeight)
	if err != nil {
		return errors.Wrap(err, "failed to query blocks")
	}

	transactionHashes, err := simpleReplicate(true, "transactions", &Transaction{}, "select * from transactions where included_in_block_hash = ANY($1)", pq.Array(blockHashes))
	if err != nil {
		return errors.Wrap(err, "failed to replicate transactions")
	}

	receiptIDs, err := simpleReplicate(true, "receipts", &Receipt{}, "select * from receipts where included_in_block_hash = ANY($1)", pq.Array(blockHashes))
	if err != nil {
		return errors.Wrap(err, "failed to replicate receipts")
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

	simpleReplicateParallel("access_keys", &AccessKey{}, "select * from access_keys where last_update_block_height > $1", baseHeight)

	simpleReplicateParallel("accounts", &Account{}, "select * from accounts where last_update_block_height > $1", baseHeight)

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
		return lastError
	}

	err = loader.Close()
	if err != nil {
		return errors.Wrap(err, "failed to finalize the load")
	}

	return nil
}
