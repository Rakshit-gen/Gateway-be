package handlers

import (
	"encoding/json"
	"gateway/internal/middleware"
	"gateway/internal/models"
	"gateway/internal/services"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type CacheRuleHandler struct {
	service      *services.CacheRuleService
	cacheService *services.CacheService
}

func NewCacheRuleHandler(service *services.CacheRuleService, cacheService *services.CacheService) *CacheRuleHandler {
	return &CacheRuleHandler{
		service:      service,
		cacheService: cacheService,
	}
}

func (h *CacheRuleHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDContextKey).(string)
	if !ok || userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req models.CreateCacheRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	rule, err := h.service.Create(r.Context(), userID, &req)
	if err != nil {
		http.Error(w, `{"error":"failed to create cache rule"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(rule)
}

func (h *CacheRuleHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDContextKey).(string)
	if !ok || userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	rules, err := h.service.List(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"failed to list cache rules"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rules)
}

func (h *CacheRuleHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDContextKey).(string)
	if !ok || userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid cache rule ID"}`, http.StatusBadRequest)
		return
	}

	var req struct {
		TTLSeconds int  `json:"ttl_seconds"`
		Enabled    bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	rule, err := h.service.Update(r.Context(), userID, id, req.TTLSeconds, req.Enabled)
	if err != nil {
		http.Error(w, `{"error":"failed to update cache rule"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rule)
}

func (h *CacheRuleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDContextKey).(string)
	if !ok || userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, `{"error":"invalid cache rule ID"}`, http.StatusBadRequest)
		return
	}

	if err := h.service.Delete(r.Context(), userID, id); err != nil {
		http.Error(w, `{"error":"failed to delete cache rule"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *CacheRuleHandler) Invalidate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Pattern string `json:"pattern"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Pattern == "" {
		req.Pattern = "cache:*"
	}

	if err := h.cacheService.Delete(r.Context(), req.Pattern); err != nil {
		http.Error(w, `{"error":"failed to invalidate cache"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "cache invalidated"})
}
