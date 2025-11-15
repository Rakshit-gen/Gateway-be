package services

import (
	"context"
	"fmt"
	"gateway/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RouteService struct {
	db *pgxpool.Pool
}

func NewRouteService(db *pgxpool.Pool) *RouteService {
	return &RouteService{db: db}
}

func (s *RouteService) Create(ctx context.Context, req *models.CreateRouteRequest) (*models.Route, error) {
	if req.LoadBalancingStrategy == "" {
		req.LoadBalancingStrategy = "round-robin"
	}
	if req.TimeoutMs == 0 {
		req.TimeoutMs = 30000
	}

	route := &models.Route{}
	err := s.db.QueryRow(
		ctx,
		`INSERT INTO routes (path, backend_urls, load_balancing_strategy, timeout_ms, retry_count)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, path, backend_urls, load_balancing_strategy, timeout_ms, retry_count, created_at`,
		req.Path, req.BackendURLs, req.LoadBalancingStrategy, req.TimeoutMs, req.RetryCount,
	).Scan(&route.ID, &route.Path, &route.BackendURLs, &route.LoadBalancingStrategy, &route.TimeoutMs, &route.RetryCount, &route.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create route: %w", err)
	}

	return route, nil
}

func (s *RouteService) GetByPath(ctx context.Context, path string) (*models.Route, error) {
	route := &models.Route{}
	err := s.db.QueryRow(
		ctx,
		`SELECT id, path, backend_urls, load_balancing_strategy, timeout_ms, retry_count, created_at
		 FROM routes WHERE path = $1`,
		path,
	).Scan(&route.ID, &route.Path, &route.BackendURLs, &route.LoadBalancingStrategy, &route.TimeoutMs, &route.RetryCount, &route.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get route: %w", err)
	}

	return route, nil
}

func (s *RouteService) GetByID(ctx context.Context, id int64) (*models.Route, error) {
	route := &models.Route{}
	err := s.db.QueryRow(
		ctx,
		`SELECT id, path, backend_urls, load_balancing_strategy, timeout_ms, retry_count, created_at
		 FROM routes WHERE id = $1`,
		id,
	).Scan(&route.ID, &route.Path, &route.BackendURLs, &route.LoadBalancingStrategy, &route.TimeoutMs, &route.RetryCount, &route.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get route: %w", err)
	}

	return route, nil
}

func (s *RouteService) List(ctx context.Context) ([]*models.Route, error) {
	rows, err := s.db.Query(
		ctx,
		`SELECT id, path, backend_urls, load_balancing_strategy, timeout_ms, retry_count, created_at
		 FROM routes ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list routes: %w", err)
	}
	defer rows.Close()

	routes := []*models.Route{}
	for rows.Next() {
		route := &models.Route{}
		if err := rows.Scan(&route.ID, &route.Path, &route.BackendURLs, &route.LoadBalancingStrategy, &route.TimeoutMs, &route.RetryCount, &route.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan route: %w", err)
		}
		routes = append(routes, route)
	}

	return routes, nil
}

func (s *RouteService) Update(ctx context.Context, id int64, req *models.UpdateRouteRequest) (*models.Route, error) {
	route := &models.Route{}
	err := s.db.QueryRow(
		ctx,
		`UPDATE routes 
		 SET backend_urls = $1, load_balancing_strategy = $2, timeout_ms = $3, retry_count = $4
		 WHERE id = $5
		 RETURNING id, path, backend_urls, load_balancing_strategy, timeout_ms, retry_count, created_at`,
		req.BackendURLs, req.LoadBalancingStrategy, req.TimeoutMs, req.RetryCount, id,
	).Scan(&route.ID, &route.Path, &route.BackendURLs, &route.LoadBalancingStrategy, &route.TimeoutMs, &route.RetryCount, &route.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to update route: %w", err)
	}

	return route, nil
}

func (s *RouteService) Delete(ctx context.Context, id int64) error {
	_, err := s.db.Exec(ctx, `DELETE FROM routes WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete route: %w", err)
	}
	return nil
}
