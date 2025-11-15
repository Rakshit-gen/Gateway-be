CREATE TABLE IF NOT EXISTS routes (
    id BIGSERIAL PRIMARY KEY,
    path VARCHAR(500) NOT NULL UNIQUE,
    backend_urls TEXT[] NOT NULL,
    load_balancing_strategy VARCHAR(50) NOT NULL DEFAULT 'round-robin',
    timeout_ms INTEGER NOT NULL DEFAULT 30000,
    retry_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_routes_path ON routes(path);

CREATE TABLE IF NOT EXISTS api_keys (
    id BIGSERIAL PRIMARY KEY,
    key VARCHAR(100) NOT NULL UNIQUE,
    name VARCHAR(200) NOT NULL,
    tier VARCHAR(50) NOT NULL DEFAULT 'free',
    rate_limit_rpm INTEGER NOT NULL DEFAULT 60,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_api_keys_key ON api_keys(key);
CREATE INDEX idx_api_keys_enabled ON api_keys(enabled);

CREATE TABLE IF NOT EXISTS cache_rules (
    id BIGSERIAL PRIMARY KEY,
    route_id BIGINT NOT NULL REFERENCES routes(id) ON DELETE CASCADE,
    ttl_seconds INTEGER NOT NULL DEFAULT 300,
    cache_key_pattern VARCHAR(200) NOT NULL DEFAULT '*',
    enabled BOOLEAN NOT NULL DEFAULT true,
    UNIQUE(route_id)
);

CREATE INDEX idx_cache_rules_route_id ON cache_rules(route_id);

CREATE TABLE IF NOT EXISTS analytics_events (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW(),
    route_id BIGINT REFERENCES routes(id) ON DELETE SET NULL,
    api_key_id BIGINT REFERENCES api_keys(id) ON DELETE SET NULL,
    status_code INTEGER NOT NULL,
    latency_ms BIGINT NOT NULL,
    cache_hit BOOLEAN NOT NULL DEFAULT false,
    ip_address VARCHAR(45) NOT NULL
);

CREATE INDEX idx_analytics_events_timestamp ON analytics_events(timestamp DESC);
CREATE INDEX idx_analytics_events_route_id ON analytics_events(route_id);
CREATE INDEX idx_analytics_events_api_key_id ON analytics_events(api_key_id);
CREATE INDEX idx_analytics_events_status_code ON analytics_events(status_code);
