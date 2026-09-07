// Package database opens the pool and applies the migrations. It knows about
// Postgres. No other package does.
package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Pool limits. A single app pod does not need more, and an unbounded pool
// turns a traffic spike into "too many clients" on the Postgres side.
const (
	maxOpenConns    = 25
	maxIdleConns    = 5
	connMaxLifetime = 30 * time.Minute
	pingInterval    = time.Second
)

// Open connects and retries until the context expires. Postgres accepts TCP
// before it accepts queries, so the first Ping after a cold start fails.
func Open(ctx context.Context, dsn string, timeout time.Duration) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(pingInterval)
	defer ticker.Stop()
	for {
		if err = db.PingContext(ctx); err == nil {
			return db, nil
		}
		slog.Warn("database not ready, retrying", "error", err)
		select {
		case <-ctx.Done():
			db.Close()
			return nil, fmt.Errorf("database unreachable after %s: %w", timeout, errors.Join(err, ctx.Err()))
		case <-ticker.C:
		}
	}
}
