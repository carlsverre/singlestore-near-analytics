import os

__config = {}


def __load(name, default=None):
    __config[name] = os.getenv(name, default)


def __getattr__(attr):
    val = __config.get(attr, None)
    if val is None:
        raise Exception("Environment variable {} is required".format(attr))
    return val


__load("POSTGRES_HOST")
__load("POSTGRES_PORT", default="5432")
__load("POSTGRES_USER")
__load("POSTGRES_PASSWORD", default="")
__load("POSTGRES_SCHEMA")
__load("POSTGRES_DB")

__load("MEMSQL_HOST")
__load("MEMSQL_PORT", default="3306")
__load("MEMSQL_USER")
__load("MEMSQL_PW", default="")
__load("MEMSQL_DB")

__load("TABLES", default="")
