# sample-app

Demo application for the [reference platform](https://github.com/reference-platform/docs).

A small notes app. A request goes from the browser to `web`, then to `api`,
then to Postgres. A background `worker` reads the notes from the api on a timer.

| Service | Language | Role | Folder |
|---|---|---|---|
| web | Go | HTML front end, port 8080 | `web/` |
| api | Python (FastAPI) | HTTP API, port 8000 | `api/` |
| db | Postgres | Storage | official image |
| worker | Go | Background loop, probes on port 8080 | `worker/` |
| migrate | dbmate | Applies the SQL migrations once, before the api starts | `db/` |

## Run

Requirements: Docker with Compose.

```
docker compose up --build
```

Open `http://localhost:8081`, add a note and check it with
`curl -s localhost:8000/api/notes`. `docker compose down` stops the stack and
keeps the data. Add `-v` to remove the data too.

## Endpoints

api:

| Endpoint | What it does |
|---|---|
| `POST /api/notes` | Stores a note. Body: `{"text": "hello"}`. |
| `GET /api/notes` | Lists the newest notes. Optional `?limit=` (1 to 200). |

web:

| Endpoint | What it does |
|---|---|
| `GET /` | HTML page with the newest 20 notes and a form. Returns 502 if the api is down. |
| `POST /notes` | Form post. Sends the note to the api, then redirects to `/`. |

worker: no business endpoints. It counts the newest notes (up to 200) every
`INTERVAL` and logs the number.

Every service has two probe endpoints:

| Endpoint | api and web | worker |
|---|---|---|
| `GET /healthz` | The process is up. Checks nothing else. | The loop ran within the last three intervals. |
| `GET /readyz` | The dependency answers: Postgres for the api, the api for web. | The last api call succeeded within the last three intervals. |

Both return 503 when the check fails.

## Configuration

All settings are environment variables.

| Variable | Service | Default | Meaning |
|---|---|---|---|
| `DATABASE_URL` | api | none, required | Postgres connection string |
| `API_URL` | web, worker | none, required | Base URL of the api |
| `INTERVAL` | worker | `30s` | Time between runs, for example `10s` or `1m` |
| `PORT` | all | api `8000`, web and worker `8080` | Listen port |
| `LOG_LEVEL` | api | `INFO` | Log level |

Logs are JSON, one object per line, on stdout.

## Develop

Requirements: Python 3.15 for the api, Go 1.27 for web and worker, and a Unix
shell (on Windows, use WSL). The tests need no database and no other service.

api:

```
cd api
python3 -m venv .venv
. .venv/bin/activate
pip install -r requirements-dev.txt
ruff check .
ruff format --check .
python -m pytest
```

web and worker (run the same commands in `worker/`):

```
cd web
gofmt -l .
go vet ./...
go test -race ./...
```

`gofmt -l .` prints nothing when the code is formatted.

To run a service from source, start the database with its schema and point the
service at it:

```
docker compose up -d migrate
DATABASE_URL=postgresql://sample:sample@localhost:5432/sample python -m app   # in api/
API_URL=http://localhost:8000 PORT=8081 go run .                              # in web/
```

## Database changes

The schema lives in `db/migrations/` as plain SQL files, one per change, with an
`up` and a `down` part. The `migrate` image applies them with
[dbmate](https://github.com/amacneil/dbmate) and exits. The api never changes
the schema. It starts without waiting for Postgres and answers `/readyz` with
503 until the database is reachable.

## Images

All service images run as a non-root user. The web and worker images are built
from `scratch` and hold one static binary.

## CI

GitHub Actions runs on every push and pull request: lint and tests for the api,
`gofmt`, `go vet` and tests for web and worker, then a test build of every
image for `linux/amd64`.

Each service has its own version. On a push to `main`, a service is released
only if a commit touched its folder since its last tag. The version comes from
the conventional commit messages: `feat` raises the minor number, `fix` and
other types raise the patch number, and `BREAKING CHANGE` raises the major
number. A commit that contains `[skip-release]` is ignored.

For a released service the workflow builds the image for `linux/amd64` and
`linux/arm64`, pushes it to GitHub Container Registry with two tags, the version
and the full git SHA, and then creates the git tag after the push:

```
ghcr.io/reference-platform/sample-app-api:1.4.0
ghcr.io/reference-platform/sample-app-api:<sha>
git tag: api-v1.4.0
```

The services are `api`, `web`, `worker` and `migrate` (built from `db/`).
Check the platforms of an image with:

```
docker buildx imagetools inspect ghcr.io/reference-platform/sample-app-web:<version>
```

All rights reserved. See [LICENSE](LICENSE).
