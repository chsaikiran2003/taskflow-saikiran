package tasks

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/saikiran/taskflow/internal/middleware"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusNotFound, "not found", nil)
		return
	}

	statusFilter := r.URL.Query().Get("status")
	assigneeFilter := r.URL.Query().Get("assignee")

	taskList, err := h.service.List(r.Context(), projectID, statusFilter, assigneeFilter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "could not fetch tasks", nil)
		return
	}
	respond(w, http.StatusOK, map[string]any{"tasks": taskList})
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusNotFound, "not found", nil)
		return
	}

	userID := middleware.GetUserID(r.Context())
	var req CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", nil)
		return
	}

	task, fields := h.service.Create(r.Context(), projectID, userID, req)
	if fields != nil {
		respondValidationError(w, fields)
		return
	}
	respond(w, http.StatusCreated, task)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusNotFound, "not found", nil)
		return
	}

	userID := middleware.GetUserID(r.Context())
	var req UpdateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", nil)
		return
	}

	task, err := h.service.Update(r.Context(), taskID, userID, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(w, http.StatusNotFound, "not found", nil)
			return
		}
		if errors.Is(err, ErrForbidden) {
			respondError(w, http.StatusForbidden, "forbidden", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "could not update task", nil)
		return
	}
	respond(w, http.StatusOK, task)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	taskID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		respondError(w, http.StatusNotFound, "not found", nil)
		return
	}

	userID := middleware.GetUserID(r.Context())
	if err := h.service.Delete(r.Context(), taskID, userID); err != nil {
		if errors.Is(err, ErrNotFound) {
			respondError(w, http.StatusNotFound, "not found", nil)
			return
		}
		if errors.Is(err, ErrForbidden) {
			respondError(w, http.StatusForbidden, "forbidden", nil)
			return
		}
		respondError(w, http.StatusInternalServerError, "could not delete task", nil)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string, fields map[string]string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	body := map[string]any{"error": message}
	if fields != nil {
		body["fields"] = fields
	}
	json.NewEncoder(w).Encode(body)
}

func respondValidationError(w http.ResponseWriter, fields map[string]string) {
	respondError(w, http.StatusBadRequest, "validation failed", fields)
}
