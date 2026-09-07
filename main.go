package main

import (
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed web
var webFS embed.FS

const maxTitleLen = 200

type Task struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

var db *sql.DB

func main() {
	dsn := env("DATABASE_URL", "postgres://devops:devops@localhost:5432/devops?sslmode=disable")
	port := env("PORT", "8080")

	var err error
	db, err = sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	if err := waitForDB(); err != nil {
		log.Fatalf("connect db: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS tasks (
		id    BIGSERIAL PRIMARY KEY,
		title TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 200),
		done  BOOLEAN NOT NULL DEFAULT false
	)`); err != nil {
		log.Fatalf("create table: %v", err)
	}

	static, _ := fs.Sub(webFS, "web")
	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServer(http.FS(static)))
	mux.HandleFunc("GET /healthz", handleHealth)
	mux.HandleFunc("GET /api/tasks", listTasks)
	mux.HandleFunc("POST /api/tasks", createTask)
	mux.HandleFunc("PUT /api/tasks/{id}", updateTask)
	mux.HandleFunc("DELETE /api/tasks/{id}", deleteTask)

	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

// waitForDB retries because the Postgres container accepts TCP before it
// accepts queries. Without this the app crash-loops on a cold `compose up`.
func waitForDB() error {
	var err error
	for i := 0; i < 30; i++ {
		if err = db.Ping(); err == nil {
			return nil
		}
		time.Sleep(time.Second)
	}
	return err
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := db.Ping(); err != nil {
		http.Error(w, "db unreachable", http.StatusServiceUnavailable)
		return
	}
	w.Write([]byte("ok"))
}

func listTasks(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, title, done FROM tasks ORDER BY id`)
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	defer rows.Close()

	tasks := []Task{}
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.ID, &t.Title, &t.Done); err != nil {
			fail(w, http.StatusInternalServerError, err)
			return
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func createTask(w http.ResponseWriter, r *http.Request) {
	t, err := decodeTask(w, r)
	if err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	var id int64
	if err := db.QueryRow(
		`INSERT INTO tasks (title, done) VALUES ($1, $2) RETURNING id`,
		t.Title, t.Done,
	).Scan(&id); err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	t.ID = id
	writeJSON(w, http.StatusCreated, t)
}

func updateTask(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	t, err := decodeTask(w, r)
	if err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	res, err := db.Exec(
		`UPDATE tasks SET title = $1, done = $2 WHERE id = $3`,
		t.Title, t.Done, id,
	)
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	t.ID = id
	if n, _ := res.RowsAffected(); n == 0 {
		fail(w, http.StatusNotFound, errors.New("no such task"))
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func deleteTask(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		fail(w, http.StatusBadRequest, err)
		return
	}
	res, err := db.Exec(`DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		fail(w, http.StatusInternalServerError, err)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		fail(w, http.StatusNotFound, errors.New("no such task"))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		return 0, errors.New("id must be a number")
	}
	return id, nil
}

func decodeTask(w http.ResponseWriter, r *http.Request) (Task, error) {
	var t Task
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&t); err != nil {
		return t, errors.New("invalid json body")
	}
	t.Title = strings.TrimSpace(t.Title)
	if t.Title == "" || len(t.Title) > maxTitleLen {
		return t, errors.New("title must be 1 to 200 characters")
	}
	return t, nil
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, err error) {
	if code >= 500 {
		log.Printf("error: %v", err)
		err = errors.New("internal error")
	}
	writeJSON(w, code, map[string]string{"error": err.Error()})
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
