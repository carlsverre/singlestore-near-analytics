import os
import pg8000
import pymysql
import textwrap
import tempfile
import subprocess
from contextlib import contextmanager

from common import config


def wait_and_check(proc):
    retcode = proc.wait()
    if retcode != 0:
        raise subprocess.CalledProcessError(retcode, proc.args)


def psql(*extra_args, **kwargs):
    args = [
        "psql",
        "--host",
        config.POSTGRES_HOST,
        "--port",
        config.POSTGRES_PORT,
        "--username",
        config.POSTGRES_USER,
        "--dbname",
        config.POSTGRES_DB,
    ]

    kwargs["env"] = os.environ.copy()
    kwargs["env"]["PGPASSWORD"] = config.POSTGRES_PASSWORD

    return subprocess.Popen(args + list(extra_args), **kwargs)


def memsql(*extra_args, **kwargs):
    args = [
        "mysql",
        "--host",
        config.MEMSQL_HOST,
        "--port",
        config.MEMSQL_PORT,
        "--user",
        config.MEMSQL_USER,
        "--database",
        config.MEMSQL_DB,
    ]

    if config.MEMSQL_PW != "":
        args.append("--password={}".format(config.MEMSQL_PW))

    return subprocess.Popen(args + list(extra_args), **kwargs)


@contextmanager
def named_pipe(suffix):
    # normally this function is unsafe to use - but since we are running as a single-use job in
    # a docker container it's fine.
    try:
        name = tempfile.mktemp(suffix=suffix)
        os.mkfifo(name)
        yield name
    finally:
        if os.path.exists(name):
            os.remove(name)


def replicate_table(src_name, src_columns, dest_name, dest_columns, bool_columns=[]):
    compression = config.COMPRESSION
    suffix = "." + compression if compression != "none" else ""

    with named_pipe(suffix) as pipename:
        dump_query = textwrap.dedent(
            """\
                \COPY (SELECT {columns} FROM {table_name}) TO STDOUT \
                WITH(FORMAT CSV, DELIMITER '\t')
            """.format(
                columns=", ".join(src_columns), table_name=src_name
            )
        )

        # Postgres boolean values need to be decoded to tinyint(1).
        # "f","t" -> 0,1
        #
        dest_columns_with_vars = map(
            lambda c: "@"+c if c in bool_columns else c, dest_columns)
        set_clause = "set " + ", ".join([c+' = decode(@'+c+', "t", 1, "f", 0)' for c in bool_columns]) \
            if len(bool_columns) > 0 else ""

        load_query = textwrap.dedent(
            """\
                LOAD DATA LOCAL INFILE '{pipename}'
                INTO TABLE {table_name} ({columns})
                COLUMNS TERMINATED BY '\\t'
                OPTIONALLY ENCLOSED BY '\\"'
                {set_clause}
            """.format(
                pipename=pipename, columns=", ".join(dest_columns_with_vars), table_name=dest_name, set_clause=set_clause
            )
        )

        # start loading from the pipe into MemSQL
        # we need to do this first since we are using a named pipe
        dest = memsql("--local-infile", "-e", load_query)

        # open the pipe in binary mode for writing only
        pipe = open(pipename, "wb")

        # start dumping into the pipe from Postgres
        src = psql("-c", dump_query, stdout=subprocess.PIPE)

        # pipe the output of Postgres into compression layer
        if compression == "lz4":
            src_gz = subprocess.Popen("lz4", stdin=src.stdout, stdout=pipe)
        elif compression == "gz":
            src_gz = subprocess.Popen("gzip", stdin=src.stdout, stdout=pipe)
        else:
            src_gz = subprocess.Popen("cat", stdin=src.stdout, stdout=pipe)

        # wait for postgres and then compression to finish
        wait_and_check(src)
        wait_and_check(src_gz)

        # close the pipe
        pipe.close()

        # wait for MemSQL to finish loading from the pipe
        wait_and_check(dest)


def connect_memsql():
    return pymysql.connect(
        host=config.MEMSQL_HOST,
        port=int(config.MEMSQL_PORT),
        user=config.MEMSQL_USER,
        password=config.MEMSQL_PW,
        db=config.MEMSQL_DB,
    )


def connect_pg():
    return pg8000.connect(
        host=config.POSTGRES_HOST,
        port=int(config.POSTGRES_PORT),
        user=config.POSTGRES_USER,
        password=config.POSTGRES_PASSWORD,
        database=config.POSTGRES_DB,
    )


def query_columns(conn, catalog, schema, name, is_memsql=False):
    with conn.cursor() as cursor:
        cursor.execute(
            """
            select column_name
            from information_schema.columns
            where
                table_catalog = %s
                AND table_schema = %s
                AND table_name = %s
                {}
            order by ordinal_position asc
        """.format(
                'AND extra != "computed"' if is_memsql else ""
            ),
            (catalog, schema, name),
        )
        rows = cursor.fetchall()
        for row in rows:
            if type(row) is dict:
                yield row["column_name"]
            else:
                yield row[0]


def bool_columns(conn, catalog, schema, name, is_memsql=False):
    with conn.cursor() as cursor:
        cursor.execute(
            """
            select column_name
            from information_schema.columns
            where
                table_catalog = %s
                AND table_schema = %s
                AND table_name = %s
                AND column_type = %s
                {}
            order by ordinal_position asc
        """.format(
                'AND extra != "computed"' if is_memsql else ""
            ),
            (catalog, schema, name, "tinyint(1)" if is_memsql else "boolean"),
        )
        rows = cursor.fetchall()
        for row in rows:
            if type(row) is dict:
                yield row["column_name"]
            else:
                yield row[0]


def query_row_count(conn, table):
    with conn.cursor() as cursor:
        cursor.execute("select count(*) as count from {}".format(table))
        row = cursor.fetchone()
        if type(row) is dict:
            return row["count"]
        else:
            return row[0]


def truncate_table(conn, table):
    with conn.cursor() as cursor:
        cursor.execute("truncate table {}".format(table))


def delete_from_table(conn, table):
    with conn.cursor() as cursor:
        cursor.execute("delete from {}".format(table))


def analyze_table(conn, table):
    with conn.cursor() as cursor:
        cursor.execute("analyze table {}".format(table))
