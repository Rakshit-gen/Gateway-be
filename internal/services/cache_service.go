package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheService struct {
	client *redis.Client
}

func NewCacheService(client *redis.Client) *CacheService {
	return &CacheService{client: client}
}

func (c *CacheService) Get(ctx context.Context, key string) (string, bool, error) {
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("failed to get cache: %w", err)
	}

	return val, true, nil
}

func (c *CacheService) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	if err := c.client.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}
	return nil
}

func (c *CacheService) Delete(ctx context.Context, pattern string) error {
	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("failed to delete cache key: %w", err)
		}
	}
	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to iterate cache keys: %w", err)
	}
	return nil
}

func (c *CacheService) GenerateKey(path, method, body string) string {
	hash := sha256.Sum256([]byte(path + method + body))
	return "cache:" + hex.EncodeToString(hash[:])
}
