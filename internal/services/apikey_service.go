package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"gateway/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type APIKeyService struct {
	db *pgxpool.Pool
}

func NewAPIKeyService(db *pgxpool.Pool) *APIKeyService {
	return &APIKeyService{db: db}
}

func (s *APIKeyService) Create(ctx context.Context, userID string, req *models.CreateAPIKeyRequest) (*models.APIKey, error) {
	key, err := generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	if req.Tier == "" {
		req.Tier = "free"
	}

	apiKey := &models.APIKey{}
	err = s.db.QueryRow(
		ctx,
		`INSERT INTO api_keys (key, name, tier, rate_limit_rpm, enabled, user_id)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, key, name, tier, rate_limit_rpm, enabled, user_id, created_at`,
		key, req.Name, req.Tier, req.RateLimitRPM, true, userID,
	).Scan(&apiKey.ID, &apiKey.Key, &apiKey.Name, &apiKey.Tier, &apiKey.RateLimitRPM, &apiKey.Enabled, &apiKey.UserID, &apiKey.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	return apiKey, nil
}

func (s *APIKeyService) GetByKey(ctx context.Context, key string) (*models.APIKey, error) {
	apiKey := &models.APIKey{}
	err := s.db.QueryRow(
		ctx,
		`SELECT id, key, name, tier, rate_limit_rpm, enabled, user_id, created_at
		 FROM api_keys WHERE key = $1 AND enabled = true`,
		key,
	).Scan(&apiKey.ID, &apiKey.Key, &apiKey.Name, &apiKey.Tier, &apiKey.RateLimitRPM, &apiKey.Enabled, &apiKey.UserID, &apiKey.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return apiKey, nil
}

func (s *APIKeyService) List(ctx context.Context, userID string) ([]*models.APIKey, error) {
	rows, err := s.db.Query(
		ctx,
		`SELECT id, key, name, tier, rate_limit_rpm, enabled, user_id, created_at
		 FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	keys := []*models.APIKey{}
	for rows.Next() {
		key := &models.APIKey{}
		if err := rows.Scan(&key.ID, &key.Key, &key.Name, &key.Tier, &key.RateLimitRPM, &key.Enabled, &key.UserID, &key.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		keys = append(keys, key)
	}

	return keys, nil
}

func (s *APIKeyService) Revoke(ctx context.Context, userID string, id int64) error {
	result, err := s.db.Exec(ctx, `UPDATE api_keys SET enabled = false WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("API key not found or access denied")
	}
	return nil
}

func (s *APIKeyService) Delete(ctx context.Context, userID string, id int64) error {
	result, err := s.db.Exec(ctx, `DELETE FROM api_keys WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("API key not found or access denied")
	}
	return nil
}

func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "gw_" + base64.URLEncoding.EncodeToString(b)[:43], nil
}
