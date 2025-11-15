package services

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client *redis.Client
}

func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

func (r *RateLimiter) Allow(ctx context.Context, key string, limit int) (bool, error) {
	now := time.Now()
	windowStart := now.Truncate(time.Minute)
	redisKey := fmt.Sprintf("ratelimit:%s:%d", key, windowStart.Unix())

	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, redisKey)
	pipe.Expire(ctx, redisKey, 2*time.Minute)
	
	if _, err := pipe.Exec(ctx); err != nil {
		return false, fmt.Errorf("failed to execute pipeline: %w", err)
	}

	count := incr.Val()
	return count <= int64(limit), nil
}

func (r *RateLimiter) GetCount(ctx context.Context, key string) (int64, error) {
	now := time.Now()
	windowStart := now.Truncate(time.Minute)
	redisKey := fmt.Sprintf("ratelimit:%s:%d", key, windowStart.Unix())

	val, err := r.client.Get(ctx, redisKey).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("failed to get count: %w", err)
	}

	return val, nil
}
