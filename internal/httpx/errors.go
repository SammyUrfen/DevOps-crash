// Package httpx holds the transport concerns that every handler shares: the
// error type a service returns, the JSON envelope, and the middleware.
package httpx

import (
	"errors"
	"fmt"
	"net/http"
)

// Error is the one error type a service returns when it wants to control the
// status code and the message the client sees. Anything else becomes a 500
// with a generic message, so an internal detail never reaches a client.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`

	status int
	cause  error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.cause }

// Status reports the HTTP status for err, and 500 for any error that is not an
// *Error.
func Status(err error) int {
	var e *Error
	if errors.As(err, &e) {
		return e.status
	}
	return http.StatusInternalServerError
}

func BadRequest(code, msg string) *Error {
	return &Error{Code: code, Message: msg, status: http.StatusBadRequest}
}

func NotFound(code, msg string) *Error {
	return &Error{Code: code, Message: msg, status: http.StatusNotFound}
}

// Internal wraps a cause that the client must not see. Details stay in the log.
func Internal(cause error) *Error {
	return &Error{
		Code:    "internal_error",
		Message: "internal error",
		status:  http.StatusInternalServerError,
		cause:   cause,
	}
}

func ServiceUnavailable(code, msg string) *Error {
	return &Error{Code: code, Message: msg, status: http.StatusServiceUnavailable}
}
