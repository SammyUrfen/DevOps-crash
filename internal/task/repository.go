package task

import (
	"context"
	"database/sql"
	"errors"

	"devops-crash/internal/httpx"
)

// Repository is the only type in this package that writes SQL.
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

const columns = `id, title, done, created_at, updated_at`

func (r *Repository) List(ctx context.Context) ([]Task, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+columns+` FROM tasks ORDER BY id`)
	if err != nil {
		return nil, httpx.Internal(err)
	}
	defer rows.Close()

	// Start from an empty slice, not nil, so the JSON body is [] and not null.
	tasks := []Task{}
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			return nil, httpx.Internal(err)
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, httpx.Internal(err)
	}
	return tasks, nil
}

func (r *Repository) Create(ctx context.Context, in Input) (Task, error) {
	t, err := scan(r.db.QueryRowContext(ctx,
		`INSERT INTO tasks (title, done) VALUES ($1, $2) RETURNING `+columns,
		in.Title, in.Done))
	if err != nil {
		return Task{}, httpx.Internal(err)
	}
	return t, nil
}

func (r *Repository) Update(ctx context.Context, id int64, in Input) (Task, error) {
	t, err := scan(r.db.QueryRowContext(ctx,
		`UPDATE tasks SET title = $1, done = $2, updated_at = now()
		 WHERE id = $3 RETURNING `+columns,
		in.Title, in.Done, id))
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return Task{}, errNotFound()
	case err != nil:
		return Task{}, httpx.Internal(err)
	}
	return t, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return httpx.Internal(err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return httpx.Internal(err)
	}
	if n == 0 {
		return errNotFound()
	}
	return nil
}

// scanner covers both *sql.Row and *sql.Rows, so one scan function serves
// every query in this file.
type scanner interface {
	Scan(dest ...any) error
}

func scan(s scanner) (Task, error) {
	var t Task
	err := s.Scan(&t.ID, &t.Title, &t.Done, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func errNotFound() error {
	return httpx.NotFound("task_not_found", "no task has that id")
}
