package handlers

import (
	"gateway/internal/analytics"
	"gateway/internal/middleware"
	"gateway/internal/models"
	"gateway/internal/services"
	"io"
	"net/http"
	"strings"
	"time"
)

type ProxyHandler struct {
	routeService     *services.RouteService
	proxyService     *services.ProxyService
	cacheService     *services.CacheService
	cacheRuleService *services.CacheRuleService
	analytics        *analytics.Analytics
}

func NewProxyHandler(
	routeService *services.RouteService,
	proxyService *services.ProxyService,
	cacheService *services.CacheService,
	cacheRuleService *services.CacheRuleService,
	analytics *analytics.Analytics,
) *ProxyHandler {
	return &ProxyHandler{
		routeService:     routeService,
		proxyService:     proxyService,
		cacheService:     cacheService,
		cacheRuleService: cacheRuleService,
		analytics:        analytics,
	}
}

func (h *ProxyHandler) Forward(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	apiKey, _ := r.Context().Value(middleware.APIKeyContextKey).(*models.APIKey)

	path := r.URL.Path
	route, err := h.routeService.GetByPath(r.Context(), path)
	if err != nil {
		for _, prefix := range []string{"/api/", "/v1/", "/v2/"} {
			if strings.HasPrefix(path, prefix) {
				testPath := strings.TrimPrefix(path, prefix)
				if testRoute, testErr := h.routeService.GetByPath(r.Context(), "/"+testPath); testErr == nil {
					route = testRoute
					break
				}
			}
		}
	}

	if route == nil {
		http.Error(w, `{"error":"route not found"}`, http.StatusNotFound)
		h.trackEvent(nil, apiKey, http.StatusNotFound, startTime, false, r.RemoteAddr)
		return
	}

	body, _ := io.ReadAll(r.Body)
	cacheKey := h.cacheService.GenerateKey(r.URL.Path, r.Method, string(body))
	cacheRule, _ := h.cacheRuleService.GetByRouteID(r.Context(), route.ID)

	if r.Method == "GET" && cacheRule != nil && cacheRule.Enabled {
		if cached, hit, err := h.cacheService.Get(r.Context(), cacheKey); err == nil && hit {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(cached))
			h.trackEvent(&route.ID, apiKey, http.StatusOK, startTime, true, r.RemoteAddr)
			return
		}
	}

	resp, err := h.proxyService.Forward(
		r.Context(),
		route.BackendURLs,
		r.Method,
		r.URL.Path,
		route.Path,
		r.Header,
		body,
		route.TimeoutMs,
	)

	if err != nil {
		http.Error(w, `{"error":"backend request failed"}`, http.StatusBadGateway)
		h.trackEvent(&route.ID, apiKey, http.StatusBadGateway, startTime, false, r.RemoteAddr)
		return
	}

	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if r.Method == "GET" && cacheRule != nil && cacheRule.Enabled && resp.StatusCode == http.StatusOK {
		ttl := time.Duration(cacheRule.TTLSeconds) * time.Second
		h.cacheService.Set(r.Context(), cacheKey, string(respBody), ttl)
	}

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.Header().Set("X-Cache", "MISS")
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)

	h.trackEvent(&route.ID, apiKey, resp.StatusCode, startTime, false, r.RemoteAddr)
}

func (h *ProxyHandler) trackEvent(routeID *int64, apiKey *models.APIKey, statusCode int, startTime time.Time, cacheHit bool, ipAddr string) {
	var apiKeyID *int64
	var userID string
	if apiKey != nil {
		apiKeyID = &apiKey.ID
		userID = apiKey.UserID
	}

	event := &models.AnalyticsEvent{
		Timestamp:  time.Now(),
		RouteID:    routeID,
		APIKeyID:   apiKeyID,
		UserID:     userID,
		StatusCode: statusCode,
		LatencyMs:  time.Since(startTime).Milliseconds(),
		CacheHit:   cacheHit,
		IPAddress:  strings.Split(ipAddr, ":")[0],
	}

	h.analytics.TrackRequest(event)
}
