package handlers

import (
	"encoding/json"
	"gateway/internal/models"
	"gateway/internal/services"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type RouteHandler struct {
	service *services.RouteService
}

func NewRouteHandler(service *services.RouteService) *RouteHandler {
	return &RouteHandler{service: service}
}

func (h *RouteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req models.CreateRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	route, err := h.service.Create(r.Context(), &req)
	if err != nil {
		http.Error(w, `{"error":"failed to create route"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(route)
}

func (h *RouteHandler) List(w http.ResponseWriter, r *http.Request) {
	routes, err := h.service.List(r.Context())
	if err != nil {
		http.Error(w, `{"error":"failed to list routes"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(routes)
}

func (h *RouteHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid route ID"}`, http.StatusBadRequest)
		return
	}

	route, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"route not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(route)
}

func (h *RouteHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid route ID"}`, http.StatusBadRequest)
		return
	}

	var req models.UpdateRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	route, err := h.service.Update(r.Context(), id, &req)
	if err != nil {
		http.Error(w, `{"error":"failed to update route"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(route)
}

func (h *RouteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid route ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		http.Error(w, `{"error":"failed to delete route"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
