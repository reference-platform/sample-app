from datetime import UTC, datetime

from fastapi.testclient import TestClient

from app.main import create_app


class FakeNotes:
    def __init__(self, healthy=True):
        self.healthy = healthy
        self.rows = []

    def open(self):
        pass

    def close(self):
        pass

    def ping(self):
        return self.healthy

    def add(self, text):
        row = {"id": len(self.rows) + 1, "text": text, "created_at": datetime.now(UTC)}
        self.rows.append(row)
        return row

    def list(self, limit):
        return list(reversed(self.rows))[:limit]


def client(notes):
    return TestClient(create_app(notes))


def test_healthz_does_not_need_the_database():
    with client(FakeNotes(healthy=False)) as c:
        assert c.get("/healthz").status_code == 200


def test_readyz_follows_the_database():
    with client(FakeNotes(healthy=True)) as c:
        assert c.get("/readyz").status_code == 200
    with client(FakeNotes(healthy=False)) as c:
        assert c.get("/readyz").status_code == 503


def test_add_and_list_notes():
    with client(FakeNotes()) as c:
        created = c.post("/api/notes", json={"text": "first"})
        assert created.status_code == 201
        assert created.json()["text"] == "first"
        c.post("/api/notes", json={"text": "second"})
        texts = [n["text"] for n in c.get("/api/notes").json()]
        assert texts == ["second", "first"]


def test_empty_note_is_rejected():
    with client(FakeNotes()) as c:
        assert c.post("/api/notes", json={"text": ""}).status_code == 422
