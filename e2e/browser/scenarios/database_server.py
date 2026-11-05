import os
import subprocess

BACKED_UP_DATABASE = "e2e_backed_up"
RESTORE_TARGET_DATABASE = "e2e_restore_target"
ROWS_TABLE = "e2e_rows"
SERVER_USER = "e2e"
SERVER_HOST_ON_COMPOSE_NETWORK = "databases"
SERVER_PORT = "5432"


def execute_sql(database: str, statement: str) -> list[str]:
    completed = subprocess.run(
        [
            "docker", "exec", _get_container_name(),
            "psql", "--username", SERVER_USER, "--dbname", database,
            "--no-align", "--tuples-only", "--set", "ON_ERROR_STOP=1", "--command", statement,
        ],
        capture_output=True,
        text=True,
    )
    assert completed.returncode == 0, f"psql failed on {database}: {completed.stderr}"

    return completed.stdout.splitlines()


def prepare_databases(rows: list[str]) -> None:
    values = ", ".join(f"('{row}')" for row in rows)

    execute_sql(BACKED_UP_DATABASE, f"CREATE TABLE {ROWS_TABLE} (id serial PRIMARY KEY, label text NOT NULL)")
    execute_sql(BACKED_UP_DATABASE, f"INSERT INTO {ROWS_TABLE} (label) VALUES {values}")
    execute_sql(BACKED_UP_DATABASE, f"CREATE DATABASE {RESTORE_TARGET_DATABASE}")


def read_rows(database: str) -> list[str]:
    return execute_sql(database, f"SELECT label FROM {ROWS_TABLE} ORDER BY id")


def _get_container_name() -> str:
    return f"{os.environ['E2E_COMPOSE_PROJECT']}-databases-1"
