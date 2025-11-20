package main

import (
	"context"
	"fmt"
	"gateway/internal/analytics"
	"gateway/internal/config"
	"gateway/internal/handlers"
	"gateway/internal/middleware"
	"gateway/internal/services"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	cfg := config.Load()

	ctx := context.Background()

	db, err := config.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	redisClient, err := config.NewRedisClient(cfg.RedisURL, cfg.RedisToken)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	routeService := services.NewRouteService(db)
	apiKeyService := services.NewAPIKeyService(db)
	cacheRuleService := services.NewCacheRuleService(db)
	rateLimiter := services.NewRateLimiter(redisClient)
	cacheService := services.NewCacheService(redisClient)
	proxyService := services.NewProxyService()
	analyticsService := analytics.NewAnalytics(db)

	routeHandler := handlers.NewRouteHandler(routeService)
	apiKeyHandler := handlers.NewAPIKeyHandler(apiKeyService)
	cacheRuleHandler := handlers.NewCacheRuleHandler(cacheRuleService, cacheService)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)
	proxyHandler := handlers.NewProxyHandler(routeService, proxyService, cacheService, cacheRuleService, analyticsService)

	analyticsCtx, cancelAnalytics := context.WithCancel(ctx)
	defer cancelAnalytics()
	go analyticsService.Start(analyticsCtx)

	r := chi.NewRouter()

	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "X-RateLimit-Limit", "X-Cache"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"Apex Gateway API","version":"1.0","endpoints":{"health":"/health","admin":"/admin/*","docs":"https://github.com/your-repo"}}`))
	})

	clerkAuth := middleware.NewClerkAuth(cfg.ClerkJWKSURL)

	r.Route("/admin", func(r chi.Router) {
		r.Use(clerkAuth.Middleware())
		r.Post("/routes", routeHandler.Create)
		r.Get("/routes", routeHandler.List)
		r.Get("/routes/{id}", routeHandler.Get)
		r.Put("/routes/{id}", routeHandler.Update)
		r.Delete("/routes/{id}", routeHandler.Delete)

		r.Post("/api-keys", apiKeyHandler.Create)
		r.Get("/api-keys", apiKeyHandler.List)
		r.Post("/api-keys/{id}/revoke", apiKeyHandler.Revoke)
		r.Delete("/api-keys/{id}", apiKeyHandler.Delete)

		r.Post("/cache-rules", cacheRuleHandler.Create)
		r.Get("/cache-rules", cacheRuleHandler.List)
		r.Put("/cache-rules/{id}", cacheRuleHandler.Update)
		r.Delete("/cache-rules/{id}", cacheRuleHandler.Delete)
		r.Post("/cache/invalidate", cacheRuleHandler.Invalidate)

		r.Get("/analytics/metrics", analyticsHandler.GetMetrics)
		r.Get("/analytics/stream", analyticsHandler.StreamMetrics)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.APIKeyAuth(apiKeyService))
		r.Use(middleware.RateLimiting(rateLimiter))
		r.HandleFunc("/*", proxyHandler.Forward)
	})

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	fmt.Println("Server exited")
}
