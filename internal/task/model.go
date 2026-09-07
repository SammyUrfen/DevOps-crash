// Package task holds one feature: the model, the storage, the rules, and the
// HTTP handlers for a task. Layers stay separate inside the package, so a
// second feature does not have to reach across four shared layer packages.
package task

import (
	"strings"
	"time"
	"unicode/utf8"

	"devops-crash/internal/httpx"
)

// TitleMaxLen matches the CHECK constraint in migration 0001. Both exist on
// purpose: the database is the last defence, the service gives a clear message.
const TitleMaxLen = 200

// Task is the record as the database and the client see it.
type Task struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Input is the part of a task that a client may set. It is a separate type so
// that a client can never write an id or a timestamp.
type Input struct {
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

// Normalize trims the title. Call it before Validate.
func (in *Input) Normalize() {
	in.Title = strings.TrimSpace(in.Title)
}

// Validate reports the first problem with the input, as a 400.
func (in Input) Validate() error {
	switch {
	case in.Title == "":
		return httpx.BadRequest("title_required", "a task needs a title")
	// Count runes, not bytes. The database CHECK counts characters too.
	case utf8.RuneCountInString(in.Title) > TitleMaxLen:
		return httpx.BadRequest("title_too_long", "a title holds 200 characters at most")
	}
	return nil
}
