# DevOps Crash — task app

A Go task API with a layered design, a Postgres database in Docker, and a
plain HTML page. The app is the payload for the DevOps steps: Docker,
Kubernetes, Helm, Terraform, CI.

## Run it

```bash
make run        # starts Postgres, then the server
```

Open http://localhost:8080

```bash
make check      # gofmt, go vet, go test -race
./smoke.sh      # 14 checks against a running server
make help       # every target
```

## Layout

Package by feature, layered inside the feature. One resource lives in one
package, so a second resource does not spread across four layer packages.

```
cmd/server/main.go        Wiring and graceful shutdown. Every dependency is built here.
internal/config/          Reads the environment once, at startup.
internal/database/        Opens the pool. Applies the migrations.
internal/httpx/           The error type, the JSON envelope, the middleware.
internal/server/          The route table, the liveness and readiness probes.
internal/task/            The feature:
    model.go                  Task, Input, the validation rules
    repository.go             The only file that writes SQL          (Model)
    service.go                The rules, over a repository interface (Model)
    handler.go                HTTP in, service call out              (Controller)
migrations/               Numbered .sql files, embedded in the binary.
web/index.html            The page, embedded in the binary.          (View)
```

The request path: `handler` → `service` → `repository` → Postgres. A layer only
calls the layer below it.

## API

| Method | Path | Body | Response |
|---|---|---|---|
| GET | `/healthz` | — | 200 while the process runs. No database call |
| GET | `/readyz` | — | 200 when the database answers, 503 when it does not |
| GET | `/api/tasks` | — | `[{id,title,done,created_at,updated_at}]` |
| POST | `/api/tasks` | `{title,done}` | 201 and the new task |
| PUT | `/api/tasks/{id}` | `{title,done}` | 200 and the task, or 404 |
| DELETE | `/api/tasks/{id}` | — | 204, or 404 |

Every error uses one envelope:

```json
{"error": {"code": "title_required", "message": "a task needs a title"}}
```

| Code | Status | Cause |
|---|---|---|
| `invalid_body` | 400 | The body is not JSON, or it carries an unknown field |
| `title_required` | 400 | The title is empty after a trim |
| `title_too_long` | 400 | The title holds more than 200 characters |
| `invalid_id` | 400 | The path id is not a positive number |
| `task_not_found` | 404 | No task has that id |
| `internal_error` | 500 | The cause goes to the log, never to the client |
| `database_unreachable` | 503 | `/readyz` only |

## Design notes

- **No web framework.** Go 1.22 `net/http` routes the method and the path value.
- **No ORM.** `database/sql` and `pgx` as the driver. The SQL stays readable.
- **No migration tool.** Numbered `.sql` files run inside a transaction with
  their ledger row. A tool earns its place when a down migration is needed.
- **Two probes, not one.** Liveness must not call the database. A database
  outage must not restart every pod.
- **Graceful shutdown.** SIGTERM drains the in-flight requests for 15 seconds.
- **The repository is an interface at the service.** The tests pass a fake and
  need no database.
- **The client never sees an internal error.** `httpx.Internal` keeps the cause
  in the log.

## Environment

| Variable | Default |
|---|---|
| `DATABASE_URL` | `postgres://devops:devops@localhost:5432/devops?sslmode=disable` |
| `PORT` | `8080` |
| `LOG_LEVEL` | `info` |

## Tests

`go test -race ./...` runs 5 tests and 17 subtests in `internal/task`: the validation table,
the id passthrough, the error passthrough, the route status codes, and the
error envelope. `smoke.sh` covers the same routes against a live server.
