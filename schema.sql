drop database if exists near;
create database near;
use near;

CREATE TABLE access_keys (
    public_key TEXT NOT NULL,
    account_id TEXT NOT NULL,
    created_by_receipt_id TEXT,
    deleted_by_receipt_id TEXT,
    permission_kind TEXT NOT NULL,
    last_update_block_height DECIMAL(20,0) NOT NULL,
    PRIMARY KEY (public_key,account_id)
);

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
    SHARD (id)
);

CREATE TABLE accounts (
    id BIGINT NOT NULL,
    account_id TEXT NOT NULL,
    created_by_receipt_id TEXT,
    deleted_by_receipt_id TEXT,
    last_update_block_height DECIMAL(20,0) NOT NULL,
    PRIMARY KEY (id)
);

-- BIG TABLE
CREATE TABLE action_receipt_actions (
    receipt_id TEXT NOT NULL,
    index_in_action_receipt INT NOT NULL,
    action_kind TEXT NOT NULL,
    args JSON NOT NULL,
    PRIMARY KEY (receipt_id, index_in_action_receipt),
    SHARD (receipt_id)
);

CREATE TABLE action_receipt_input_data (
    input_data_id TEXT NOT NULL,
    input_to_receipt_id TEXT NOT NULL,
    PRIMARY KEY (input_data_id, input_to_receipt_id)
);

CREATE TABLE action_receipt_output_data (
    output_data_id TEXT NOT NULL,
    output_from_receipt_id TEXT NOT NULL,
    receiver_account_id TEXT NOT NULL,
    PRIMARY KEY (output_data_id, output_from_receipt_id)
);

CREATE TABLE action_receipts (
    receipt_id TEXT NOT NULL,
    signer_account_id TEXT NOT NULL,
    signer_public_key TEXT NOT NULL,
    gas_price DECIMAL(45,0) NOT NULL,
    PRIMARY KEY (receipt_id)
);

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
    SHARD (block_hash)
);

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

CREATE TABLE data_receipts (
    data_id TEXT NOT NULL,
    receipt_id TEXT NOT NULL,
    data LONGBLOB,
    PRIMARY KEY (data_id)
) ;

CREATE TABLE execution_outcome_receipts (
    executed_receipt_id TEXT NOT NULL,
    index_in_execution_outcome INT NOT NULL,
    produced_receipt_id TEXT NOT NULL,
    PRIMARY KEY (executed_receipt_id, index_in_execution_outcome, produced_receipt_id)
);

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

-- BIG TABLE
CREATE TABLE transaction_actions (
    transaction_hash TEXT NOT NULL,
    index_in_transaction INT NOT NULL,
    action_kind TEXT NOT NULL,
    args JSON NOT NULL,
    PRIMARY KEY (transaction_hash, index_in_transaction)
);

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
