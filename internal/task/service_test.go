package task

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"devops-crash/internal/httpx"
)

// fakeRepo records the last input and returns what the test tells it to.
type fakeRepo struct {
	got     Input
	gotID   int64
	calls   int
	failErr error
}

func (f *fakeRepo) List(context.Context) ([]Task, error) {
	f.calls++
	return []Task{}, f.failErr
}

func (f *fakeRepo) Create(_ context.Context, in Input) (Task, error) {
	f.calls, f.got = f.calls+1, in
	return Task{ID: 1, Title: in.Title, Done: in.Done}, f.failErr
}

func (f *fakeRepo) Update(_ context.Context, id int64, in Input) (Task, error) {
	f.calls, f.got, f.gotID = f.calls+1, in, id
	return Task{ID: id, Title: in.Title, Done: in.Done}, f.failErr
}

func (f *fakeRepo) Delete(_ context.Context, id int64) error {
	f.calls, f.gotID = f.calls+1, id
	return f.failErr
}

func TestCreateValidates(t *testing.T) {
	cases := []struct {
		name       string
		title      string
		wantStatus int
		wantTitle  string
	}{
		{"plain title", "buy milk", 0, "buy milk"},
		{"trims the title", "  buy milk  ", 0, "buy milk"},
		{"rejects an empty title", "", http.StatusBadRequest, ""},
		{"rejects only spaces", "   ", http.StatusBadRequest, ""},
		{"accepts 200 characters", strings.Repeat("a", 200), 0, strings.Repeat("a", 200)},
		{"rejects 201 characters", strings.Repeat("a", 201), http.StatusBadRequest, ""},
		// A rune is not a byte. 200 emoji are 800 bytes and still a valid title.
		{"counts runes not bytes", strings.Repeat("😀", 200), 0, strings.Repeat("😀", 200)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepo{}
			svc := NewService(repo)

			got, err := svc.Create(context.Background(), Input{Title: tc.title})

			if tc.wantStatus != 0 {
				if err == nil {
					t.Fatalf("want an error, got task %+v", got)
				}
				if s := httpx.Status(err); s != tc.wantStatus {
					t.Errorf("status: want %d, got %d", tc.wantStatus, s)
				}
				if repo.calls != 0 {
					t.Errorf("invalid input reached the repository")
				}
				return
			}
			if err != nil {
				t.Fatalf("want no error, got %v", err)
			}
			if repo.got.Title != tc.wantTitle {
				t.Errorf("title sent to the repository: want %q, got %q", tc.wantTitle, repo.got.Title)
			}
		})
	}
}

func TestUpdateValidatesAndPassesTheID(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewService(repo)

	if _, err := svc.Update(context.Background(), 42, Input{Title: " done "}); err != nil {
		t.Fatalf("want no error, got %v", err)
	}
	if repo.gotID != 42 {
		t.Errorf("id: want 42, got %d", repo.gotID)
	}
	if repo.got.Title != "done" {
		t.Errorf("title: want %q, got %q", "done", repo.got.Title)
	}

	if _, err := svc.Update(context.Background(), 42, Input{Title: ""}); err == nil {
		t.Error("want an error for an empty title")
	}
}

func TestRepositoryErrorPassesThrough(t *testing.T) {
	want := httpx.Internal(errors.New("connection reset"))
	svc := NewService(&fakeRepo{failErr: want})

	_, err := svc.Create(context.Background(), Input{Title: "ok"})
	if !errors.Is(err, error(want)) {
		t.Fatalf("want the repository error, got %v", err)
	}
	if httpx.Status(err) != http.StatusInternalServerError {
		t.Errorf("status: want 500, got %d", httpx.Status(err))
	}
}
