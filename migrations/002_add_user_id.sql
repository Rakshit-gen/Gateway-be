-- Add user_id column to routes table (nullable first, then we'll make it NOT NULL for new records)
ALTER TABLE routes ADD COLUMN IF NOT EXISTS user_id VARCHAR(255);
CREATE INDEX IF NOT EXISTS idx_routes_user_id ON routes(user_id);
-- Note: Existing records will have NULL user_id. New records must have user_id set.

-- Add user_id column to api_keys table
ALTER TABLE api_keys ADD COLUMN IF NOT EXISTS user_id VARCHAR(255);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
-- Note: Existing records will have NULL user_id. New records must have user_id set.

-- Add user_id column to cache_rules table
ALTER TABLE cache_rules ADD COLUMN IF NOT EXISTS user_id VARCHAR(255);
CREATE INDEX IF NOT EXISTS idx_cache_rules_user_id ON cache_rules(user_id);
-- Note: Existing records will have NULL user_id. New records must have user_id set.

-- Add user_id column to analytics_events table
ALTER TABLE analytics_events ADD COLUMN IF NOT EXISTS user_id VARCHAR(255);
CREATE INDEX IF NOT EXISTS idx_analytics_events_user_id ON analytics_events(user_id);
-- Note: Analytics events can have NULL user_id for legacy records

