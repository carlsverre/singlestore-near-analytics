drop database if exists near;
create database near;
use near;

CREATE TABLE `access_keys` (
    `public_key` text NOT NULL,
    `account_id` text NOT NULL,
    `created_by_receipt_id` text,
    `deleted_by_receipt_id` text,
    `permission_kind` text NOT NULL,
    `last_update_block_height` decimal(20,0) NOT NULL,
    PRIMARY KEY (`public_key`,`account_id`)
) ;

-- BIG TABLE
CREATE TABLE `account_changes` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT,
    `affected_account_id` text NOT NULL,
    `changed_in_block_timestamp` decimal(20,0) NOT NULL,
    `changed_in_block_hash` text NOT NULL,
    `caused_by_transaction_hash` text,
    `caused_by_receipt_id` text,
    `update_reason` text NOT NULL,
    `affected_account_nonstaked_balance` decimal(45,0) NOT NULL,
    `affected_account_staked_balance` decimal(45,0) NOT NULL,
    `affected_account_storage_usage` decimal(20,0) NOT NULL,
    KEY (`id`) using clustered columnstore,
    SHARD (`id`)
) ;

CREATE TABLE `accounts` (
    `id` bigint(20) NOT NULL AUTO_INCREMENT,
    `account_id` text NOT NULL,
    `created_by_receipt_id` text,
    `deleted_by_receipt_id` text,
    `last_update_block_height` decimal(20,0) NOT NULL,
    PRIMARY KEY (`id`)
) ;

-- BIG TABLE
/*
CREATE TABLE `action_receipt_actions` (
    `receipt_id` text NOT NULL,
    `index_in_action_receipt` int(11) NOT NULL,
    `action_kind` text NOT NULL,
    `args` text NOT NULL,
    KEY (`receipt_id`,`index_in_action_receipt`) using clustered columnstore,
    SHARD (`receipt_id`)
) ;
*/
CREATE TABLE `action_receipt_actions` (
    `receipt_id` text NOT NULL,
    `index_in_action_receipt` int(11) NOT NULL,
    `action_kind` text NOT NULL,
    `args` text NOT NULL,
    primary KEY (`receipt_id`,`index_in_action_receipt`),
    SHARD (`receipt_id`)
) ;

CREATE TABLE `action_receipt_input_data` (
    `input_data_id` text NOT NULL,
    `input_to_receipt_id` text NOT NULL,
    PRIMARY KEY (`input_data_id`,`input_to_receipt_id`)
) ;

CREATE TABLE `action_receipt_output_data` (
    `output_data_id` text NOT NULL,
    `output_from_receipt_id` text NOT NULL,
    `receiver_account_id` text NOT NULL,
    PRIMARY KEY (`output_data_id`,`output_from_receipt_id`)
) ;

CREATE TABLE `action_receipts` (
    `receipt_id` text NOT NULL,
    `signer_account_id` text NOT NULL,
    `signer_public_key` text NOT NULL,
    `gas_price` decimal(45,0) NOT NULL,
    PRIMARY KEY (`receipt_id`)
) ;

-- BIG TABLE
CREATE TABLE `blocks` (
    `block_height` decimal(20,0) NOT NULL,
    `block_hash` text NOT NULL,
    `prev_block_hash` text NOT NULL,
    `block_timestamp` decimal(20,0) NOT NULL,
    `total_supply` decimal(45,0) NOT NULL,
    `gas_price` decimal(45,0) NOT NULL,
    `author_account_id` text NOT NULL,
    KEY (`block_hash`) using clustered columnstore,
    SHARD (`block_hash`)
) ;

-- BIG TABLE
CREATE TABLE `chunks` (
    `included_in_block_hash` text NOT NULL,
    `chunk_hash` text NOT NULL,
    `shard_id` decimal(20,0) NOT NULL,
    `signature` text NOT NULL,
    `gas_limit` decimal(20,0) NOT NULL,
    `gas_used` decimal(20,0) NOT NULL,
    `author_account_id` text NOT NULL,
    KEY (`chunk_hash`) using clustered columnstore,
    SHARD (chunk_hash)
) ;

CREATE TABLE `data_receipts` (
    `data_id` text NOT NULL,
    `receipt_id` text NOT NULL,
    `data` longblob,
    primary KEY (`data_id`)
) ;

CREATE TABLE `execution_outcome_receipts` (
    `executed_receipt_id` text NOT NULL,
    `index_in_execution_outcome` int(11) NOT NULL,
    `produced_receipt_id` text NOT NULL,
    primary KEY (`executed_receipt_id`,`index_in_execution_outcome`,`produced_receipt_id`)
);

CREATE TABLE `execution_outcomes` (
    `receipt_id` text NOT NULL,
    `executed_in_block_hash` text NOT NULL,
    `executed_in_block_timestamp` decimal(20,0) NOT NULL,
    `executed_in_chunk_hash` text NOT NULL,
    `index_in_chunk` int(11) NOT NULL,
    `gas_burnt` decimal(20,0) NOT NULL,
    `tokens_burnt` decimal(45,0) NOT NULL,
    `executor_account_id` text NOT NULL,
    `status` text NOT NULL,
    primary KEY (`receipt_id`)
);

CREATE TABLE `receipts` (
    `receipt_id` text NOT NULL,
    `included_in_block_hash` text NOT NULL,
    `included_in_chunk_hash` text NOT NULL,
    `index_in_chunk` int(11) NOT NULL,
    `included_in_block_timestamp` decimal(20,0) NOT NULL,
    `predecessor_account_id` text NOT NULL,
    `receiver_account_id` text NOT NULL,
    `receipt_kind` text NOT NULL,
    `originated_from_transaction_hash` text NOT NULL,
    primary KEY (`receipt_id`)
);

-- BIG TABLE
CREATE TABLE `transaction_actions` (
    `transaction_hash` text NOT NULL,
    `index_in_transaction` int(11) NOT NULL,
    `action_kind` text NOT NULL,
    `args` text NOT NULL,
    key (`transaction_hash`,`index_in_transaction`) using clustered columnstore,
    shard(`transaction_hash`)
);

CREATE TABLE `transactions` (
    `transaction_hash` text NOT NULL,
    `included_in_block_hash` text NOT NULL,
    `included_in_chunk_hash` text NOT NULL,
    `index_in_chunk` int(11) NOT NULL,
    `block_timestamp` decimal(20,0) NOT NULL,
    `signer_account_id` text NOT NULL,
    `signer_public_key` text NOT NULL,
    `nonce` decimal(20,0) NOT NULL,
    `receiver_account_id` text NOT NULL,
    `signature` text NOT NULL,
    `status` text NOT NULL,
    `converted_into_receipt_id` text NOT NULL,
    `receipt_conversion_gas_burnt` decimal(20,0) DEFAULT NULL,
    `receipt_conversion_tokens_burnt` decimal(45,0) DEFAULT NULL,
    primary KEY (`transaction_hash`)
);
