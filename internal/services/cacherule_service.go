package services

import (
	"context"
	"fmt"
	"gateway/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CacheRuleService struct {
	db *pgxpool.Pool
}

func NewCacheRuleService(db *pgxpool.Pool) *CacheRuleService {
	return &CacheRuleService{db: db}
}

func (s *CacheRuleService) Create(ctx context.Context, userID string, req *models.CreateCacheRuleRequest) (*models.CacheRule, error) {
	if req.CacheKeyPattern == "" {
		req.CacheKeyPattern = "*"
	}

	// Verify that the route belongs to the user
	var routeUserID string
	err := s.db.QueryRow(ctx, `SELECT user_id FROM routes WHERE id = $1`, req.RouteID).Scan(&routeUserID)
	if err != nil {
		return nil, fmt.Errorf("route not found: %w", err)
	}
	if routeUserID != userID {
		return nil, fmt.Errorf("access denied: route does not belong to user")
	}

	rule := &models.CacheRule{}
	err = s.db.QueryRow(
		ctx,
		`INSERT INTO cache_rules (route_id, ttl_seconds, cache_key_pattern, enabled, user_id)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, route_id, ttl_seconds, cache_key_pattern, enabled, user_id`,
		req.RouteID, req.TTLSeconds, req.CacheKeyPattern, true, userID,
	).Scan(&rule.ID, &rule.RouteID, &rule.TTLSeconds, &rule.CacheKeyPattern, &rule.Enabled, &rule.UserID)

	if err != nil {
		return nil, fmt.Errorf("failed to create cache rule: %w", err)
	}

	return rule, nil
}

func (s *CacheRuleService) GetByRouteID(ctx context.Context, routeID int64) (*models.CacheRule, error) {
	rule := &models.CacheRule{}
	err := s.db.QueryRow(
		ctx,
		`SELECT id, route_id, ttl_seconds, cache_key_pattern, enabled, user_id
		 FROM cache_rules WHERE route_id = $1 AND enabled = true`,
		routeID,
	).Scan(&rule.ID, &rule.RouteID, &rule.TTLSeconds, &rule.CacheKeyPattern, &rule.Enabled, &rule.UserID)

	if err != nil {
		return nil, fmt.Errorf("failed to get cache rule: %w", err)
	}

	return rule, nil
}

func (s *CacheRuleService) List(ctx context.Context, userID string) ([]*models.CacheRule, error) {
	rows, err := s.db.Query(
		ctx,
		`SELECT id, route_id, ttl_seconds, cache_key_pattern, enabled, user_id
		 FROM cache_rules WHERE user_id = $1 ORDER BY id DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list cache rules: %w", err)
	}
	defer rows.Close()

	rules := []*models.CacheRule{}
	for rows.Next() {
		rule := &models.CacheRule{}
		if err := rows.Scan(&rule.ID, &rule.RouteID, &rule.TTLSeconds, &rule.CacheKeyPattern, &rule.Enabled, &rule.UserID); err != nil {
			return nil, fmt.Errorf("failed to scan cache rule: %w", err)
		}
		rules = append(rules, rule)
	}

	return rules, nil
}

func (s *CacheRuleService) Update(ctx context.Context, userID string, id int64, ttl int, enabled bool) (*models.CacheRule, error) {
	rule := &models.CacheRule{}
	err := s.db.QueryRow(
		ctx,
		`UPDATE cache_rules SET ttl_seconds = $1, enabled = $2 WHERE id = $3 AND user_id = $4
		 RETURNING id, route_id, ttl_seconds, cache_key_pattern, enabled, user_id`,
		ttl, enabled, id, userID,
	).Scan(&rule.ID, &rule.RouteID, &rule.TTLSeconds, &rule.CacheKeyPattern, &rule.Enabled, &rule.UserID)

	if err != nil {
		return nil, fmt.Errorf("failed to update cache rule: %w", err)
	}

	return rule, nil
}

func (s *CacheRuleService) Delete(ctx context.Context, userID string, id int64) error {
	result, err := s.db.Exec(ctx, `DELETE FROM cache_rules WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete cache rule: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("cache rule not found or access denied")
	}
	return nil
}
