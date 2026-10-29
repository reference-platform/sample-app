# sample-app

Demo application for the [reference platform](https://github.com/reference-platform/docs).

## api

A small HTTP API in Python (FastAPI) that stores short text notes in Postgres.
It gives the platform a realistic workload: a service with a database,
health probes, structured logs and a container image.

A request to `POST /api/notes` is validated, then written to the `notes` table
through a small pool of database connections. `GET /api/notes` reads the newest
rows. The two probe endpoints let an orchestrator tell "the process is up"
(`/healthz`) from "the process can serve traffic" (`/readyz`, which needs the
database).

| Endpoint | What it does |
|---|---|
| `GET /healthz` | The process is up. Does not touch the database. |
| `GET /readyz` | The database answers. Returns 503 if it does not. |
| `POST /api/notes` | Stores a note. Body: `{"text": "hello"}`. |
| `GET /api/notes` | Lists the newest notes. Optional `?limit=` (1 to 200). |

### Configuration

All settings come from environment variables.

| Variable | Default | Meaning |
|---|---|---|
| `DATABASE_URL` | none, required | Postgres connection string |
| `PORT` | `8000` | Listen port |
| `LOG_LEVEL` | `INFO` | Log level |

Logs are JSON, one object per line, on stdout.

### Run it locally

Requirements:

- Docker
- Python 3.15
- A Unix shell. On Windows, use WSL.

Start Postgres:

```
docker network create sample
docker run -d --name sample-pg --network sample \
  -e POSTGRES_USER=sample -e POSTGRES_PASSWORD=sample -e POSTGRES_DB=sample \
  -p 127.0.0.1:5432:5432 \
  postgres:18.6-trixie
```

Install, lint and test:

```
cd api
python3 -m venv .venv
. .venv/bin/activate
pip install -r requirements-dev.txt
ruff check .
ruff format --check .
python -m pytest
```

Run the service:

```
DATABASE_URL=postgresql://sample:sample@localhost:5432/sample python -m app
```

Try it from another terminal:

```
curl -s localhost:8000/healthz
curl -si localhost:8000/readyz
curl -s -X POST localhost:8000/api/notes -H 'content-type: application/json' -d '{"text":"hello"}'
curl -s localhost:8000/api/notes
```

### Run it in a container

```
docker build -t sample-app-api:dev api
docker run --rm --network sample -p 127.0.0.1:8000:8000 \
  -e DATABASE_URL=postgresql://sample:sample@sample-pg:5432/sample \
  sample-app-api:dev
```

Clean up:

```
docker rm -f sample-pg
docker network rm sample
```

### Limits

- The `notes` table is created at startup. A real deployment uses a migration
  tool or a migration job.
- The service waits up to 30 seconds for the database at startup, then exits.
  The orchestrator restarts it.

All rights reserved. See [LICENSE](LICENSE).
