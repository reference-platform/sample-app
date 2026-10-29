import logging
import os
from contextlib import asynccontextmanager
from datetime import datetime

from fastapi import FastAPI, HTTPException, Request
from pydantic import BaseModel, Field

from app.db import Notes, PostgresNotes

log = logging.getLogger("api")


class NoteIn(BaseModel):
    text: str = Field(min_length=1, max_length=500)


class NoteOut(BaseModel):
    id: int
    text: str
    created_at: datetime


def create_app(notes: Notes | None = None) -> FastAPI:
    @asynccontextmanager
    async def lifespan(app: FastAPI):
        app.state.notes = notes or PostgresNotes(os.environ["DATABASE_URL"])
        app.state.notes.open()
        yield
        app.state.notes.close()

    app = FastAPI(title="api", lifespan=lifespan)

    @app.get("/healthz")
    def healthz():
        return {"status": "ok"}

    @app.get("/readyz")
    def readyz(request: Request):
        if not request.app.state.notes.ping():
            raise HTTPException(status_code=503, detail="database not ready")
        return {"status": "ok"}

    @app.post("/api/notes", status_code=201, response_model=NoteOut)
    def add_note(note: NoteIn, request: Request):
        row = request.app.state.notes.add(note.text)
        log.info("note created, id=%s", row["id"])
        return row

    @app.get("/api/notes", response_model=list[NoteOut])
    def list_notes(request: Request, limit: int = 50):
        return request.app.state.notes.list(min(max(limit, 1), 200))

    return app
