package analytics

import (
	"context"
	"fmt"
	"gateway/internal/models"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Analytics struct {
	db      *pgxpool.Pool
	eventCh chan *models.AnalyticsEvent
}

func NewAnalytics(db *pgxpool.Pool) *Analytics {
	return &Analytics{
		db:      db,
		eventCh: make(chan *models.AnalyticsEvent, 1000),
	}
}

func (a *Analytics) TrackRequest(event *models.AnalyticsEvent) {
	select {
	case a.eventCh <- event:
	default:
		// Queue full, drop event silently
	}
}

func (a *Analytics) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	events := make([]*models.AnalyticsEvent, 0, 100)

	for {
		select {
		case <-ctx.Done():
			a.flushEvents(context.Background(), events)
			return

		case event := <-a.eventCh:
			events = append(events, event)
			if len(events) >= 100 {
				a.flushEvents(ctx, events)
				events = events[:0]
			}

		case <-ticker.C:
			if len(events) > 0 {
				a.flushEvents(ctx, events)
				events = events[:0]
			}
		}
	}
}

func (a *Analytics) flushEvents(ctx context.Context, events []*models.AnalyticsEvent) {
	if len(events) == 0 {
		return
	}

	batch := &pgx.Batch{}
	for _, event := range events {
		batch.Queue(
			`INSERT INTO analytics_events 
			(timestamp, route_id, api_key_id, status_code, latency_ms, cache_hit, ip_address)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			event.Timestamp, event.RouteID, event.APIKeyID,
			event.StatusCode, event.LatencyMs, event.CacheHit, event.IPAddress,
		)
	}

	br := a.db.SendBatch(ctx, batch)
	defer br.Close()

	for range events {
		if _, err := br.Exec(); err != nil {
			// Optional: log error
			return
		}
	}
}

func (a *Analytics) GetMetrics(ctx context.Context, startTime, endTime time.Time) (*models.AnalyticsMetrics, error) {
	metrics := &models.AnalyticsMetrics{}

	var totalRequests, cacheHits, errors int64
	err := a.db.QueryRow(
		ctx,
		`SELECT 
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE cache_hit = true) as cache_hits,
			COUNT(*) FILTER (WHERE status_code >= 400) as errors
		 FROM analytics_events 
		 WHERE timestamp >= $1 AND timestamp <= $2`,
		startTime, endTime,
	).Scan(&totalRequests, &cacheHits, &errors)

	if err != nil {
		return nil, fmt.Errorf("failed to get basic metrics: %w", err)
	}

	metrics.TotalRequests = totalRequests
	if totalRequests > 0 {
		metrics.ErrorRate = float64(errors) / float64(totalRequests)
		metrics.CacheHitRatio = float64(cacheHits) / float64(totalRequests)
	}

	var p50, p95, p99 *int64
	err = a.db.QueryRow(
		ctx,
		`SELECT 
			PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY latency_ms) as p50,
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms) as p95,
			PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY latency_ms) as p99
		 FROM analytics_events 
		 WHERE timestamp >= $1 AND timestamp <= $2`,
		startTime, endTime,
	).Scan(&p50, &p95, &p99)

	if err == nil && p50 != nil {
		metrics.LatencyP50 = *p50
		metrics.LatencyP95 = *p95
		metrics.LatencyP99 = *p99
	}

	rows, err := a.db.Query(
		ctx,
		`SELECT 
			DATE_TRUNC('minute', timestamp) as minute,
			COUNT(*) as count
		 FROM analytics_events 
		 WHERE timestamp >= $1 AND timestamp <= $2
		 GROUP BY minute
		 ORDER BY minute DESC
		 LIMIT 60`,
		startTime, endTime,
	)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var rpm models.RequestsPerMin
			if err := rows.Scan(&rpm.Timestamp, &rpm.Count); err == nil {
				metrics.RequestsPerMin = append(metrics.RequestsPerMin, rpm)
			}
		}
	}

	endpointRows, err := a.db.Query(
		ctx,
		`SELECT 
			r.path,
			COUNT(*) as request_count,
			AVG(ae.latency_ms)::bigint as avg_latency,
			COUNT(*) FILTER (WHERE ae.status_code >= 400)::float / COUNT(*)::float as error_rate
		 FROM analytics_events ae
		 LEFT JOIN routes r ON ae.route_id = r.id
		 WHERE ae.timestamp >= $1 AND ae.timestamp <= $2 AND r.path IS NOT NULL
		 GROUP BY r.path
		 ORDER BY request_count DESC
		 LIMIT 10`,
		startTime, endTime,
	)
	if err == nil {
		defer endpointRows.Close()
		for endpointRows.Next() {
			var stat models.EndpointStats
			if err := endpointRows.Scan(&stat.Path, &stat.RequestCount, &stat.AvgLatencyMs, &stat.ErrorRate); err == nil {
				metrics.TopEndpoints = append(metrics.TopEndpoints, stat)
			}
		}
	}

	return metrics, nil
}

func (a *Analytics) GetRealtimeMetrics(ctx context.Context) (*models.AnalyticsMetrics, error) {
	now := time.Now()
	startTime := now.Add(-5 * time.Minute)
	return a.GetMetrics(ctx, startTime, now)
}
