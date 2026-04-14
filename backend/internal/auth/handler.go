package auth

import (
	"encoding/json"
	"net/http"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", nil)
		return
	}

	resp, fields := h.service.Register(r.Context(), req)
	if fields != nil {
		respondValidationError(w, fields)
		return
	}

	respond(w, http.StatusCreated, resp)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", nil)
		return
	}

	resp, fields := h.service.Login(r.Context(), req)
	if fields != nil {
		respondValidationError(w, fields)
		return
	}

	respond(w, http.StatusOK, resp)
}

// Shared response helpers
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
