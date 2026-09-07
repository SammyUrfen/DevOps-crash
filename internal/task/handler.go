package task

import (
	"net/http"
	"strconv"

	"devops-crash/internal/httpx"
)

// Handler turns HTTP into service calls. It holds no rules of its own.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Routes registers the task endpoints on mux.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/tasks", h.list)
	mux.HandleFunc("POST /api/tasks", h.create)
	mux.HandleFunc("PUT /api/tasks/{id}", h.update)
	mux.HandleFunc("DELETE /api/tasks/{id}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.svc.List(r.Context())
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, tasks)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in Input
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	t, err := h.svc.Create(r.Context(), in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, t)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	var in Input
	if err := httpx.DecodeJSON(w, r, &in); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	t, err := h.svc.Update(r.Context(), id, in)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	httpx.JSON(w, http.StatusOK, t)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		httpx.WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, httpx.BadRequest("invalid_id", "the id must be a positive number")
	}
	return id, nil
}
