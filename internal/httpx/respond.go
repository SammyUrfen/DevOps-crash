package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

// MaxBodyBytes caps a request body. A CRUD task never needs more.
const MaxBodyBytes = 4 << 10

// JSON writes v as the response body. It never fails the request after the
// status is written, so an encode error only reaches the log.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("encode response", "error", err)
	}
}

// WriteError is the single place that turns an error into a response. Every
// handler calls this, so the envelope stays the same across the API.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	status := Status(err)
	if status >= http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "request failed",
			"method", r.Method, "path", r.URL.Path, "error", err)
	}

	var e *Error
	if !errors.As(err, &e) {
		e = Internal(err)
	}
	JSON(w, status, map[string]*Error{"error": e})
}

// DecodeJSON reads a bounded JSON body and rejects any unknown field, so a
// typo in a client payload fails loudly instead of writing a zero value.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, MaxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return BadRequest("invalid_body", "the request body is not valid JSON for this endpoint")
	}
	return nil
}
