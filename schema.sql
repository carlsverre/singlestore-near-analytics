drop database if exists near;
create database near;
use near;

-- the replication_meta table contains one row per range of blocks replicated to
-- this database.  The block_height field refers to the highest block_height in
-- the range of replicated blocks.
CREATE TABLE replication_meta (
    block_height DECIMAL(20,0) NOT NULL,
    KEY (block_height) USING CLUSTERED COLUMNSTORE,
    UNIQUE KEY (block_height) USING HASH,
    SHARD (block_height)
);

CREATE TABLE access_keys (
    public_key TEXT NOT NULL,
    account_id TEXT NOT NULL,
    created_by_receipt_id TEXT,
    deleted_by_receipt_id TEXT,
    permission_kind TEXT NOT NULL,
    last_update_block_height DECIMAL(20,0) NOT NULL,
    PRIMARY KEY (public_key,account_id)
);

CREATE INDEX access_keys_account_id_idx ON access_keys  (account_id);
CREATE INDEX access_keys_last_update_block_height_idx ON access_keys  (last_update_block_height);
CREATE INDEX access_keys_public_key_idx ON access_keys  (public_key);

-- BIG TABLE
CREATE TABLE account_changes (
    id BIGINT NOT NULL,
    affected_account_id TEXT NOT NULL,
    changed_in_block_timestamp DECIMAL(20,0) NOT NULL,
    changed_in_block_hash TEXT NOT NULL,
    caused_by_transaction_hash TEXT,
    caused_by_receipt_id TEXT,
    update_reason TEXT NOT NULL,
    affected_account_nonstaked_balance DECIMAL(45,0) NOT NULL,
    affected_account_staked_balance DECIMAL(45,0) NOT NULL,
    affected_account_storage_usage DECIMAL(20,0) NOT NULL,
    KEY (id) USING CLUSTERED COLUMNSTORE,
    UNIQUE KEY (id) USING HASH,
    SHARD (id)
);

CREATE INDEX account_changes_affected_account_id_idx ON account_changes  (affected_account_id) using hash;
CREATE INDEX account_changes_changed_in_block_hash_idx ON account_changes  (changed_in_block_hash) using hash;
CREATE INDEX account_changes_changed_in_block_timestamp_idx ON account_changes  (changed_in_block_timestamp) using hash;
CREATE INDEX account_changes_changed_in_caused_by_receipt_id_idx ON account_changes  (caused_by_receipt_id) using hash;
CREATE INDEX account_changes_changed_in_caused_by_transaction_hash_idx ON account_changes  (caused_by_transaction_hash) using hash;

CREATE TABLE accounts (
    id BIGINT NOT NULL,
    account_id TEXT NOT NULL,
    created_by_receipt_id TEXT,
    deleted_by_receipt_id TEXT,
    last_update_block_height DECIMAL(20,0) NOT NULL,
    PRIMARY KEY (id)
);

CREATE INDEX accounts_last_update_block_height_idx ON accounts  (last_update_block_height);

-- BIG TABLE
CREATE TABLE action_receipt_actions (
    receipt_id TEXT NOT NULL,
    index_in_action_receipt INT NOT NULL,
    action_kind TEXT NOT NULL,
    args JSON NOT NULL,
    receipt_predecessor_account_id TEXT NOT NULL,
    receipt_receiver_account_id TEXT NOT NULL,
    receipt_included_in_block_timestamp DECIMAL(20,0) NOT NULL,
    args_method_name AS args::$method_name PERSISTED TEXT,

    PRIMARY KEY (receipt_id, index_in_action_receipt),
    SHARD (receipt_id)
);

CREATE INDEX action_receipt_actions_receipt_args_method_name ON action_receipt_actions(args_method_name);
CREATE INDEX action_receipt_actions_receipt_predecessor_account_id_idx ON action_receipt_actions(receipt_predecessor_account_id);
CREATE INDEX action_receipt_actions_receipt_receiver_account_id_idx ON action_receipt_actions(receipt_receiver_account_id);
CREATE INDEX action_receipt_actions_receipt_included_in_block_timestamp_idx ON action_receipt_actions(receipt_included_in_block_timestamp);
CREATE INDEX action_receipt_actions_action_kind_idx ON action_receipt_actions (action_kind);

CREATE TABLE action_receipt_input_data (
    input_data_id TEXT NOT NULL,
    input_to_receipt_id TEXT NOT NULL,
    PRIMARY KEY (input_data_id, input_to_receipt_id)
);

CREATE INDEX action_receipt_input_data_input_data_id_idx ON action_receipt_input_data  (input_data_id);
CREATE INDEX action_receipt_input_data_input_to_receipt_id_idx ON action_receipt_input_data  (input_to_receipt_id);

CREATE TABLE action_receipt_output_data (
    output_data_id TEXT NOT NULL,
    output_from_receipt_id TEXT NOT NULL,
    receiver_account_id TEXT NOT NULL,
    PRIMARY KEY (output_data_id, output_from_receipt_id)
);

CREATE INDEX action_receipt_output_data_output_data_id_idx ON action_receipt_output_data  (output_data_id);
CREATE INDEX action_receipt_output_data_output_from_receipt_id_idx ON action_receipt_output_data  (output_from_receipt_id);
CREATE INDEX action_receipt_output_data_receiver_account_id_idx ON action_receipt_output_data  (receiver_account_id);

CREATE TABLE action_receipts (
    receipt_id TEXT NOT NULL,
    signer_account_id TEXT NOT NULL,
    signer_public_key TEXT NOT NULL,
    gas_price DECIMAL(45,0) NOT NULL,
    PRIMARY KEY (receipt_id)
);

CREATE INDEX action_receipt_signer_account_id_idx ON action_receipts  (signer_account_id);

-- BIG TABLE
CREATE TABLE blocks (
    block_height DECIMAL(20,0) NOT NULL,
    block_hash TEXT NOT NULL,
    prev_block_hash TEXT NOT NULL,
    block_timestamp DECIMAL(20,0) NOT NULL,
    total_supply DECIMAL(45,0) NOT NULL,
    gas_price DECIMAL(45,0) NOT NULL,
    author_account_id TEXT NOT NULL,
    KEY (block_hash) USING CLUSTERED COLUMNSTORE,
    UNIQUE KEY (block_hash) USING HASH,
    SHARD (block_hash)
);

CREATE INDEX blocks_height_idx ON blocks  (block_height) using hash;
CREATE INDEX blocks_prev_hash_idx ON blocks  (prev_block_hash) using hash;
CREATE INDEX blocks_timestamp_idx ON blocks  (block_timestamp) using hash;

-- BIG TABLE
CREATE TABLE chunks (
    included_in_block_hash TEXT NOT NULL,
    chunk_hash TEXT NOT NULL,
    shard_id DECIMAL(20,0) NOT NULL,
    signature TEXT NOT NULL,
    gas_limit DECIMAL(20,0) NOT NULL,
    gas_used DECIMAL(20,0) NOT NULL,
    author_account_id TEXT NOT NULL,
    KEY (chunk_hash) USING CLUSTERED COLUMNSTORE,
    SHARD (chunk_hash)
);

CREATE INDEX chunks_included_in_block_hash_idx ON chunks  (included_in_block_hash) using hash;

CREATE TABLE data_receipts (
    data_id TEXT NOT NULL,
    receipt_id TEXT NOT NULL,
    data LONGBLOB,
    PRIMARY KEY (data_id)
) ;

CREATE INDEX data_receipts_receipt_id_idx ON data_receipts  (receipt_id);

CREATE TABLE execution_outcome_receipts (
    executed_receipt_id TEXT NOT NULL,
    index_in_execution_outcome INT NOT NULL,
    produced_receipt_id TEXT NOT NULL,
    PRIMARY KEY (executed_receipt_id, index_in_execution_outcome, produced_receipt_id)
);

CREATE INDEX execution_outcome_receipts_produced_receipt_id ON execution_outcome_receipts  (produced_receipt_id);

CREATE TABLE execution_outcomes (
    receipt_id TEXT NOT NULL,
    executed_in_block_hash TEXT NOT NULL,
    executed_in_block_timestamp DECIMAL(20,0) NOT NULL,
    index_in_chunk INT NOT NULL,
    gas_burnt DECIMAL(20,0) NOT NULL,
    tokens_burnt DECIMAL(45,0) NOT NULL,
    executor_account_id TEXT NOT NULL,
    status TEXT NOT NULL,
    shard_id DECIMAL(20,0) NOT NULL,
    PRIMARY KEY (receipt_id)
);

CREATE INDEX execution_outcomes_status_idx ON execution_outcomes (status);
CREATE INDEX execution_outcome_executed_in_block_timestamp ON execution_outcomes  (executed_in_block_timestamp);
CREATE INDEX execution_outcome_executed_in_chunk_hash_idx ON execution_outcomes  (executed_in_chunk_hash);
CREATE INDEX execution_outcomes_block_hash_idx ON execution_outcomes  (executed_in_block_hash);
CREATE INDEX execution_outcomes_receipt_id_idx ON execution_outcomes  (receipt_id);

CREATE TABLE receipts (
    receipt_id TEXT NOT NULL,
    included_in_block_hash TEXT NOT NULL,
    included_in_chunk_hash TEXT NOT NULL,
    index_in_chunk INT NOT NULL,
    included_in_block_timestamp DECIMAL(20,0) NOT NULL,
    predecessor_account_id TEXT NOT NULL,
    receiver_account_id TEXT NOT NULL,
    receipt_kind TEXT NOT NULL,
    originated_from_transaction_hash TEXT NOT NULL,
    PRIMARY KEY (receipt_id)
);

CREATE INDEX receipts_originated_from_transaction_hash_idx ON receipts (originated_from_transaction_hash);
CREATE INDEX receipts_included_in_block_hash_idx ON receipts  (included_in_block_hash);
CREATE INDEX receipts_included_in_chunk_hash_idx ON receipts  (included_in_chunk_hash);
CREATE INDEX receipts_predecessor_account_id_idx ON receipts  (predecessor_account_id);
CREATE INDEX receipts_receiver_account_id_idx ON receipts  (receiver_account_id);
CREATE INDEX receipts_timestamp_idx ON receipts  (included_in_block_timestamp);

-- BIG TABLE
CREATE TABLE transaction_actions (
    transaction_hash TEXT NOT NULL,
    index_in_transaction INT NOT NULL,
    action_kind TEXT NOT NULL,
    args JSON NOT NULL,
    KEY (transaction_hash, index_in_transaction) USING CLUSTERED COLUMNSTORE,
    SHARD (transaction_hash)
);

CREATE INDEX transactions_actions_action_kind_idx ON transaction_actions (action_kind) USING HASH;

CREATE TABLE transactions (
    transaction_hash TEXT NOT NULL,
    included_in_block_hash TEXT NOT NULL,
    included_in_chunk_hash TEXT NOT NULL,
    index_in_chunk INT NOT NULL,
    block_timestamp DECIMAL(20,0) NOT NULL,
    signer_account_id TEXT NOT NULL,
    signer_public_key TEXT NOT NULL,
    nonce DECIMAL(20,0) NOT NULL,
    receiver_account_id TEXT NOT NULL,
    signature TEXT NOT NULL,
    status TEXT NOT NULL,
    converted_into_receipt_id TEXT NOT NULL,
    receipt_conversion_gas_burnt DECIMAL(20,0) DEFAULT NULL,
    receipt_conversion_tokens_burnt DECIMAL(45,0) DEFAULT NULL,
    PRIMARY KEY (transaction_hash)
);

CREATE INDEX transactions_receiver_account_id_idx ON transactions (receiver_account_id);
CREATE INDEX transactions_converted_into_receipt_id_dx ON transactions  (converted_into_receipt_id);
CREATE INDEX transactions_included_in_block_hash_idx ON transactions  (included_in_block_hash);
CREATE INDEX transactions_included_in_block_timestamp_idx ON transactions  (block_timestamp);
CREATE INDEX transactions_included_in_chunk_hash_idx ON transactions  (included_in_chunk_hash);
CREATE INDEX transactions_signer_account_id_idx ON transactions  (signer_account_id);
CREATE INDEX transactions_signer_public_key_idx ON transactions  (signer_public_key);