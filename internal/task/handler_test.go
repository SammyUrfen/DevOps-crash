package task

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// serve runs one request against the real route table and a fake repository.
func serve(repo repository, method, path, body string) *httptest.ResponseRecorder {
	mux := http.NewServeMux()
	NewHandler(NewService(repo)).Routes(mux)

	var reader *strings.Reader = strings.NewReader(body)
	req := httptest.NewRequest(method, path, reader)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestHandlerStatusCodes(t *testing.T) {
	cases := []struct {
		name     string
		method   string
		path     string
		body     string
		want     int
		wantCode string // the code field in the error envelope
	}{
		{"list", "GET", "/api/tasks", "", http.StatusOK, ""},
		{"create", "POST", "/api/tasks", `{"title":"ok"}`, http.StatusCreated, ""},
		{"update", "PUT", "/api/tasks/7", `{"title":"ok","done":true}`, http.StatusOK, ""},
		{"delete", "DELETE", "/api/tasks/7", "", http.StatusNoContent, ""},
		{"bad json", "POST", "/api/tasks", `not json`, http.StatusBadRequest, "invalid_body"},
		{"unknown field", "POST", "/api/tasks", `{"title":"ok","id":9}`, http.StatusBadRequest, "invalid_body"},
		{"empty title", "POST", "/api/tasks", `{"title":" "}`, http.StatusBadRequest, "title_required"},
		{"id not a number", "DELETE", "/api/tasks/abc", "", http.StatusBadRequest, "invalid_id"},
		{"id is zero", "PUT", "/api/tasks/0", `{"title":"ok"}`, http.StatusBadRequest, "invalid_id"},
		{"no route", "PATCH", "/api/tasks/7", "", http.StatusMethodNotAllowed, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := serve(&fakeRepo{}, tc.method, tc.path, tc.body)
			if rec.Code != tc.want {
				t.Fatalf("status: want %d, got %d, body %s", tc.want, rec.Code, rec.Body)
			}
			if tc.wantCode == "" {
				return
			}
			var env struct {
				Error struct {
					Code    string `json:"code"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("body is not the error envelope: %s", rec.Body)
			}
			if env.Error.Code != tc.wantCode {
				t.Errorf("code: want %q, got %q", tc.wantCode, env.Error.Code)
			}
			if env.Error.Message == "" {
				t.Error("the error envelope carries no message")
			}
		})
	}
}

func TestListReturnsAnArrayNotNull(t *testing.T) {
	rec := serve(&fakeRepo{}, "GET", "/api/tasks", "")
	if got := strings.TrimSpace(rec.Body.String()); got != "[]" {
		t.Errorf("body: want [], got %s", got)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content type: want application/json, got %q", ct)
	}
}
