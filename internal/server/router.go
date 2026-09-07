// Package server assembles the routes and the middleware. It is the only place
// that knows the full URL map.
package server

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"devops-crash/internal/httpx"
	"devops-crash/internal/task"
	"devops-crash/web"
)

// readyTimeout bounds the readiness ping. A probe that hangs is a probe that
// never reports, so keep it well under the probe period.
const readyTimeout = 2 * time.Second

// Router returns the handler for the whole application.
func Router(db *sql.DB, tasks *task.Handler) http.Handler {
	mux := http.NewServeMux()

	// GET / matches every path that no other pattern claims, so the page and
	// its assets come from the embedded files.
	mux.Handle("GET /", http.FileServerFS(web.FS()))
	mux.HandleFunc("GET /healthz", live)
	mux.HandleFunc("GET /readyz", ready(db))
	tasks.Routes(mux)

	return httpx.Chain(mux, httpx.Recover, httpx.Logger)
}

// live answers as long as the process runs. Kubernetes restarts the pod when
// this fails, so it must not depend on the database.
func live(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ready answers only when the database answers. Kubernetes takes the pod out
// of the Service when this fails, and puts it back when the database returns.
func ready(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			httpx.WriteError(w, r, httpx.ServiceUnavailable("database_unreachable",
				"the database does not answer"))
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}
