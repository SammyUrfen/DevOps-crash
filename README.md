# DevOps Crash — task app

A Go CRUD API and a plain HTML page, backed by Postgres in a Docker container.
The app is the payload for the DevOps steps: Docker, Kubernetes, Helm, Terraform, CI.

## Run it

```bash
docker compose up -d          # Postgres on port 5432
go build -o bin/app . && ./bin/app
```

Open http://localhost:8080

## Verify

```bash
./smoke.sh                    # 10 checks against the API
```

## API

| Method | Path | Body | Response |
|---|---|---|---|
| GET | `/healthz` | — | `ok`, or 503 when the database is down |
| GET | `/api/tasks` | — | `[{id,title,done}]` |
| POST | `/api/tasks` | `{title,done}` | 201 + the new task |
| PUT | `/api/tasks/{id}` | `{title,done}` | 200 + the task, or 404 |
| DELETE | `/api/tasks/{id}` | — | 204, or 404 |

## Files

| Path | What it holds |
|---|---|
| `main.go` | The whole server: routes, database access, validation |
| `web/index.html` | The page. Embedded in the binary with `go:embed` |
| `docker-compose.yml` | Postgres only |
| `smoke.sh` | The test |
| `.env.example` | `DATABASE_URL` and `PORT` |

## Design notes

- No web framework. Go 1.22 `net/http` routes methods and path values.
- No migration tool. One `CREATE TABLE IF NOT EXISTS` runs at startup.
- The server retries the database for 30 seconds at startup.
  Postgres accepts TCP before it accepts queries.
- The page is embedded, so the binary is the whole deployment unit.
