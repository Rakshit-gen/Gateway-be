package models

import (
	"time"
)

type Route struct {
	ID                     int64     `json:"id"`
	Path                   string    `json:"path"`
	BackendURLs            []string  `json:"backend_urls"`
	LoadBalancingStrategy  string    `json:"load_balancing_strategy"`
	TimeoutMs              int       `json:"timeout_ms"`
	RetryCount             int       `json:"retry_count"`
	UserID                 string    `json:"user_id"`
	CreatedAt              time.Time `json:"created_at"`
}

type APIKey struct {
	ID           int64     `json:"id"`
	Key          string    `json:"key"`
	Name         string    `json:"name"`
	Tier         string    `json:"tier"`
	RateLimitRPM int       `json:"rate_limit_rpm"`
	Enabled      bool      `json:"enabled"`
	UserID       string    `json:"user_id"`
	CreatedAt    time.Time `json:"created_at"`
}

type CacheRule struct {
	ID              int64  `json:"id"`
	RouteID         int64  `json:"route_id"`
	TTLSeconds      int    `json:"ttl_seconds"`
	CacheKeyPattern string `json:"cache_key_pattern"`
	Enabled         bool   `json:"enabled"`
	UserID          string `json:"user_id"`
}

type AnalyticsEvent struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	RouteID   *int64    `json:"route_id"`
	APIKeyID  *int64    `json:"api_key_id"`
	UserID    string    `json:"user_id"`
	StatusCode int      `json:"status_code"`
	LatencyMs int64     `json:"latency_ms"`
	CacheHit  bool      `json:"cache_hit"`
	IPAddress string    `json:"ip_address"`
}

type CreateRouteRequest struct {
	Path                   string   `json:"path"`
	BackendURLs            []string `json:"backend_urls"`
	LoadBalancingStrategy  string   `json:"load_balancing_strategy"`
	TimeoutMs              int      `json:"timeout_ms"`
	RetryCount             int      `json:"retry_count"`
}

type UpdateRouteRequest struct {
	BackendURLs            []string `json:"backend_urls"`
	LoadBalancingStrategy  string   `json:"load_balancing_strategy"`
	TimeoutMs              int      `json:"timeout_ms"`
	RetryCount             int      `json:"retry_count"`
}

type CreateAPIKeyRequest struct {
	Name         string `json:"name"`
	Tier         string `json:"tier"`
	RateLimitRPM int    `json:"rate_limit_rpm"`
}

type CreateCacheRuleRequest struct {
	RouteID         int64  `json:"route_id"`
	TTLSeconds      int    `json:"ttl_seconds"`
	CacheKeyPattern string `json:"cache_key_pattern"`
}

type AnalyticsMetrics struct {
	TotalRequests  int64              `json:"total_requests"`
	ErrorRate      float64            `json:"error_rate"`
	CacheHitRatio  float64            `json:"cache_hit_ratio"`
	LatencyP50     int64              `json:"latency_p50"`
	LatencyP95     int64              `json:"latency_p95"`
	LatencyP99     int64              `json:"latency_p99"`
	RequestsPerMin []RequestsPerMin   `json:"requests_per_min"`
	TopEndpoints   []EndpointStats    `json:"top_endpoints"`
}

type RequestsPerMin struct {
	Timestamp time.Time `json:"timestamp"`
	Count     int64     `json:"count"`
}

type EndpointStats struct {
	Path          string  `json:"path"`
	RequestCount  int64   `json:"request_count"`
	AvgLatencyMs  int64   `json:"avg_latency_ms"`
	ErrorRate     float64 `json:"error_rate"`
}
