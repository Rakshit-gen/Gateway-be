package config

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Port         string
	DatabaseURL  string
	RedisURL     string
	RedisToken   string
	AllowOrigins []string
	ClerkJWKSURL string
}

func Load() *Config {
	// Support multiple origins - comma-separated list or single origin
	frontendURL := getEnv("FRONTEND_URL", "http://localhost:3000")
	allowedOriginsEnv := getEnv("ALLOWED_ORIGINS", "")
	
	var allowedOrigins []string
	if allowedOriginsEnv != "" {
		// Parse comma-separated origins
		origins := strings.Split(allowedOriginsEnv, ",")
		for _, origin := range origins {
			origin = strings.TrimSpace(origin)
			// Remove trailing slashes from origins (CORS origins should not have trailing slashes)
			origin = strings.TrimSuffix(origin, "/")
			if origin != "" {
				allowedOrigins = append(allowedOrigins, origin)
			}
		}
	} else {
		// Fallback to FRONTEND_URL (single origin)
		frontendURL = strings.TrimSuffix(frontendURL, "/")
		allowedOrigins = []string{frontendURL}
	}
	
	// Always include localhost for local development (if not in production)
	hasLocalhost := false
	for _, origin := range allowedOrigins {
		if strings.Contains(origin, "localhost") {
			hasLocalhost = true
			break
		}
	}
	// Only add localhost if we're not in a production-only environment
	if !hasLocalhost && !strings.Contains(strings.Join(allowedOrigins, ","), "vercel.app") {
		allowedOrigins = append(allowedOrigins, "http://localhost:3000")
	}

	return &Config{
		Port:         getEnv("PORT", "8080"),
		DatabaseURL:  getEnv("DATABASE_URL", ""),
		RedisURL:     getEnv("REDIS_URL", ""),
		RedisToken:   getEnv("REDIS_TOKEN", ""),
		AllowOrigins: allowedOrigins,
		ClerkJWKSURL: getEnv("CLERK_JWKS_URL", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func NewPostgresPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	config.MaxConns = 20
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

func NewRedisClient(redisURL, token string) (*redis.Client, error) {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis URL: %w", err)
	}

	if token != "" {
		opts.Password = token
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	return client, nil
}
