package httpx

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// statusRecorder captures the status code, which net/http does not expose to
// middleware after the handler writes it.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

// Logger records one line for each request, after the handler returns.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds())
	})
}

// Recover keeps one panicking handler from killing the whole server.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				slog.Error("panic in handler",
					"path", r.URL.Path, "value", v, "stack", string(debug.Stack()))
				JSON(w, http.StatusInternalServerError,
					map[string]*Error{"error": Internal(nil)})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// Chain applies middleware so the first argument runs first.
func Chain(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}
