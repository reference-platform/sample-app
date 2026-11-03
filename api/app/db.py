from typing import Protocol

import psycopg
from psycopg.rows import dict_row
from psycopg_pool import ConnectionPool


class Notes(Protocol):
    def open(self) -> None: ...
    def close(self) -> None: ...
    def ping(self) -> bool: ...
    def add(self, text: str) -> dict: ...
    def list(self, limit: int) -> list[dict]: ...


class PostgresNotes:
    def __init__(self, url: str):
        self.pool = ConnectionPool(
            url,
            min_size=1,
            max_size=5,
            timeout=5,
            open=False,
            kwargs={"row_factory": dict_row},
        )

    def open(self):
        self.pool.open()

    def close(self):
        self.pool.close()

    def ping(self):
        try:
            with self.pool.connection(timeout=2) as conn:
                conn.execute("SELECT 1")
            return True
        except (psycopg.OperationalError, psycopg.InterfaceError):
            return False

    def add(self, text):
        with self.pool.connection() as conn:
            return conn.execute(
                "INSERT INTO notes (text) VALUES (%s) RETURNING id, text, created_at",
                (text,),
            ).fetchone()

    def list(self, limit):
        with self.pool.connection() as conn:
            return conn.execute(
                "SELECT id, text, created_at FROM notes ORDER BY id DESC LIMIT %s",
                (limit,),
            ).fetchall()
