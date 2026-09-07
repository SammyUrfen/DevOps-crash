// Command server wires the application and runs it. Every dependency is built
// here and passed down. No package reads a global.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"devops-crash/internal/config"
	"devops-crash/internal/database"
	"devops-crash/internal/server"
	"devops-crash/internal/task"
	"devops-crash/migrations"
)

// Header and body read limits stop a slow client from holding a connection
// open forever.
const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout,
		&slog.HandlerOptions{Level: cfg.LogLevel})))

	// The context ends on SIGINT or SIGTERM. Kubernetes sends SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, cfg.DatabaseURL, cfg.DBConnectTimeout)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := database.Migrate(ctx, db, migrations.FS); err != nil {
		return err
	}

	handler := task.NewHandler(task.NewService(task.NewRepository(db)))
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.Router(db, handler),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	errc := make(chan error, 1)
	go func() {
		slog.Info("listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	// Drain in-flight requests before the process exits, so a rolling deploy
	// does not drop a request.
	slog.Info("shutdown started", "timeout", cfg.ShutdownTimeout.String())
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
