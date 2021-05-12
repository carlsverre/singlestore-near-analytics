import sys
import os
import logging
import time
import pymysql
from collections import namedtuple

from common import util, config

ReplicationConfig = namedtuple("ReplicationConfig", ("src", "dest"))


def __table(src, dest=None):
    if dest is None:
        return ReplicationConfig(src, src)
    else:
        return ReplicationConfig(src, dest)


TABLES = [
    __table("access_keys"),
    __table("account_changes"),
    __table("accounts"),
    __table("action_receipt_actions"),
    __table("action_receipt_input_data"),
    __table("action_receipt_output_data"),
    __table("action_receipts"),
    __table("blocks"),
    __table("chunks"),
    __table("data_receipts"),
    __table("execution_outcome_receipts"),
    __table("execution_outcomes"),
    __table("receipts"),
    __table("transaction_actions"),
    __table("transactions"),
]

keywords = [
    "reads",
    "primary"
]


def main():
    logging.basicConfig(
        level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s"
    )

    pg_conn = util.connect_pg()
    memsql_conn = util.connect_memsql()

    max_block_height = 0
    with pg_conn.cursor() as cursor:
        cursor.execute("select coalesce(max(block_height), 0) from blocks")
        row = cursor.fetchone()
        max_block_height = row[0]

    logging.info("max block height at replication start: %s" % max_block_height)

    if config.TABLES != "":
        def _t(t):
            return __table(t)
        tables = map(_t, config.TABLES.split(","))
    else:
        tables = TABLES

    compression = config.COMPRESSION
    if compression not in ("none", "lz4", "gz"):
        raise Exception("COMPRESSION must be one of (lz4, gz, none)")

    for table in tables:
        logging.info("Replicating `%s` to `%s`", table.src, table.dest)

        pg_columns = list(
            util.query_columns(
                pg_conn, config.POSTGRES_DB, config.POSTGRES_SCHEMA, table.src
            )
        )
        memsql_columns = list(
            util.query_columns(
                memsql_conn, "def", config.MEMSQL_DB, table.dest, is_memsql=True
            )
        )

        # postgres exports booleans differently than memsql ingests tinyint(1)
        # so we need the list of boolean columns to properly construct LOAD DATA statement later
        bool_columns = list(
            util.bool_columns(
                memsql_conn, "def", config.MEMSQL_DB, table.dest, is_memsql=True
            )
        )

        # filter the source columns for only the columns at the destination
        pg_columns = [col for col in pg_columns if col in memsql_columns]

        # Sort the source column set based on the order of the destination column set
        # NOTE: we could sort alphanumerically - however MemSQL must read all of
        # the columns up to the sort key for each row at the aggregator level,
        # so by ensuring it's the first column (which it is in the table
        # definitions in MemSQL) we also maximize performance.
        # sort both column sets so they match
        pg_columns = sorted(pg_columns, key=memsql_columns.index)

        pg_count = util.query_row_count(pg_conn, table.src)

        logging.info("Source count: %d", pg_count)

        assert pg_columns == memsql_columns, "source and destination columns must match"
        assert len(memsql_columns) > 0, "there must be columns to replicate"

        # keywords need to be quoted on both postgres and memsql side
        pg_columns = list(
            map(lambda x: '"'+x+'"' if x in keywords else x, pg_columns))
        logging.info("Source columns: %s", pg_columns)
        memsql_columns = list(
            map(lambda x: '`'+x+'`' if x in keywords else x, memsql_columns))
        logging.info("Destination columns: %s", memsql_columns)
        bool_columns = list(
            map(lambda x: '`'+x+'`' if x in keywords else x, bool_columns))

        # We have seen failures of replication that leave tables empty
        # so do this transactionally
        logging.info("Starting transaction")
        memsql_conn.autocommit = False
        try:
            logging.info("Truncating destination table `%s`", table.dest)
            start = time.time()
            util.delete_from_table(memsql_conn, table.dest)
            memsql_conn.commit()
            duration = time.time() - start
            logging.info("Truncated table `%s` in %0.2f seconds",
                         table.dest, duration)

            logging.info("Starting replication from `%s` to `%s`",
                         table.src, table.dest)
            start = time.time()
            util.replicate_table(table.src, pg_columns,
                                 table.dest, memsql_columns, bool_columns)
            duration = time.time() - start
            memsql_conn.commit()
            logging.info("Ended transaction")
        except pymysql.Error as error:
            logging.error("Rolled back transaction", error)
            memsql_conn.rollback()
            raise

        memsql_conn.autocommit = True

        memsql_count = util.query_row_count(memsql_conn, table.dest)
        if pg_count != memsql_count:
            print("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
            print("WARN: row counts differ post replication! postgres({}) memsql({})".format(pg_count, memsql_count))
            print("WARN: start replication stream before block_height = {}".format(max_block_height))
            print("!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")

        logging.info("Running analyze table on `%s`", table.dest)
        util.analyze_table(memsql_conn, table.dest)

        logging.info(
            "Replicated %d rows from Postgres to MemSQL in %0.2f seconds",
            memsql_count,
            duration,
        )

    logging.info("data initialization finished; start replication stream at block_height = {}".format(max_block_height))


if __name__ == "__main__":
    main()
