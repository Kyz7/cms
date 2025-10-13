-- ==========================================
-- POSTGRESQL FULL-TEXT SEARCH SETUP
-- ==========================================

-- 1. Add GIN index for JSONB data (faster JSON queries)
CREATE INDEX IF NOT EXISTS idx_content_entries_data_gin 
ON content_entries USING GIN (data);

-- 2. Add GIN index for full-text search on data
CREATE INDEX IF NOT EXISTS idx_content_entries_data_tsvector 
ON content_entries USING GIN (to_tsvector('english', data::text));

-- 3. Add composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_content_entries_type_status 
ON content_entries(content_type_id, status);

CREATE INDEX IF NOT EXISTS idx_content_entries_creator_created 
ON content_entries(created_by, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_content_entries_status_created 
ON content_entries(status, created_at DESC);

-- 4. Add index for published date queries
CREATE INDEX IF NOT EXISTS idx_content_entries_published 
ON content_entries(published_at DESC) 
WHERE published_at IS NOT NULL;

-- 5. Create function to extract searchable text from JSON
CREATE OR REPLACE FUNCTION extract_searchable_text(jsonb_data JSONB)
RETURNS TEXT AS $$
DECLARE
    result TEXT := '';
    key TEXT;
    value TEXT;
BEGIN
    FOR key, value IN SELECT * FROM jsonb_each_text(jsonb_data)
    LOOP
        IF value IS NOT NULL THEN
            result := result || ' ' || value;
        END IF;
    END LOOP;
    RETURN result;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- 6. Add computed column for full-text search (optional, for better performance)
ALTER TABLE content_entries 
ADD COLUMN IF NOT EXISTS search_vector tsvector 
GENERATED ALWAYS AS (to_tsvector('english', extract_searchable_text(data))) STORED;

-- Add GIN index on the computed column
CREATE INDEX IF NOT EXISTS idx_content_entries_search_vector 
ON content_entries USING GIN (search_vector);

-- 7. Update existing rows to populate search_vector (if column already exists)
-- This is automatic for GENERATED ALWAYS columns

-- ==========================================
-- SEARCH ANALYTICS TABLE (OPTIONAL)
-- ==========================================

CREATE TABLE IF NOT EXISTS search_logs (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id) ON DELETE SET NULL,
    query TEXT NOT NULL,
    filters JSONB,
    results_count INTEGER,
    execution_time_ms INTEGER,
    clicked_result_id INTEGER REFERENCES content_entries(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_search_logs_query ON search_logs(query);
CREATE INDEX idx_search_logs_user ON search_logs(user_id);
CREATE INDEX idx_search_logs_created ON search_logs(created_at DESC);

-- ==========================================
-- SEARCH PERFORMANCE VIEWS
-- ==========================================

-- Popular search terms
CREATE OR REPLACE VIEW popular_search_terms AS
SELECT 
    query,
    COUNT(*) as search_count,
    AVG(results_count) as avg_results,
    AVG(execution_time_ms) as avg_time_ms
FROM search_logs
WHERE created_at > NOW() - INTERVAL '30 days'
GROUP BY query
ORDER BY search_count DESC
LIMIT 100;

-- Zero result queries (need attention)
CREATE OR REPLACE VIEW zero_result_queries AS
SELECT 
    query,
    COUNT(*) as attempt_count,
    MAX(created_at) as last_attempted
FROM search_logs
WHERE results_count = 0
AND created_at > NOW() - INTERVAL '30 days'
GROUP BY query
ORDER BY attempt_count DESC
LIMIT 50;

-- ==========================================
-- CUSTOM FUNCTIONS FOR ADVANCED SEARCH
-- ==========================================

-- Function to search with field weighting (title more important than content)
CREATE OR REPLACE FUNCTION weighted_search(
    search_query TEXT,
    content_type_id INTEGER DEFAULT NULL
)
RETURNS TABLE (
    entry_id INTEGER,
    rank REAL
) AS $
BEGIN
    RETURN QUERY
    SELECT 
        ce.id::INTEGER,
        (
            ts_rank(to_tsvector('english', ce.data->>'title'), plainto_tsquery('english', search_query)) * 1.5 +
            ts_rank(to_tsvector('english', ce.data->>'content'), plainto_tsquery('english', search_query)) * 1.0 +
            ts_rank(to_tsvector('english', ce.data->>'description'), plainto_tsquery('english', search_query)) * 0.8
        )::REAL as rank
    FROM content_entries ce
    WHERE 
        (content_type_id IS NULL OR ce.content_type_id = content_type_id)
        AND to_tsvector('english', ce.data::text) @@ plainto_tsquery('english', search_query)
    ORDER BY rank DESC;
END;
$ LANGUAGE plpgsql;

-- Function to get similar entries (based on content similarity)
CREATE OR REPLACE FUNCTION find_similar_entries(
    entry_id INTEGER,
    similarity_threshold REAL DEFAULT 0.3,
    max_results INTEGER DEFAULT 10
)
RETURNS TABLE (
    similar_entry_id INTEGER,
    similarity_score REAL
) AS $
DECLARE
    base_vector tsvector;
BEGIN
    -- Get the search vector of the base entry
    SELECT search_vector INTO base_vector
    FROM content_entries
    WHERE id = entry_id;
    
    RETURN QUERY
    SELECT 
        ce.id::INTEGER,
        ts_rank(ce.search_vector, base_vector)::REAL as similarity_score
    FROM content_entries ce
    WHERE 
        ce.id != entry_id
        AND ts_rank(ce.search_vector, base_vector) > similarity_threshold
    ORDER BY similarity_score DESC
    LIMIT max_results;
END;
$ LANGUAGE plpgsql;

-- ==========================================
-- TRIGGERS FOR SEARCH OPTIMIZATION
-- ==========================================

-- Trigger to log search queries automatically (optional)
CREATE OR REPLACE FUNCTION log_search_query()
RETURNS TRIGGER AS $
BEGIN
    -- This would be called from application layer
    -- Just a placeholder for future implementation
    RETURN NEW;
END;
$ LANGUAGE plpgsql;

-- ==========================================
-- MAINTENANCE QUERIES
-- ==========================================

-- Vacuum and analyze for search performance
-- Run periodically (weekly recommended)
-- VACUUM ANALYZE content_entries;
-- REINDEX INDEX idx_content_entries_data_gin;
-- REINDEX INDEX idx_content_entries_search_vector;

-- ==========================================
-- USAGE EXAMPLES
-- ==========================================

-- Example 1: Basic full-text search
-- SELECT * FROM content_entries
-- WHERE to_tsvector('english', data::text) @@ plainto_tsquery('english', 'golang programming');

-- Example 2: Search with ranking
-- SELECT id, data->>'title', 
--        ts_rank(to_tsvector('english', data::text), plainto_tsquery('english', 'golang')) as rank
-- FROM content_entries
-- WHERE to_tsvector('english', data::text) @@ plainto_tsquery('english', 'golang')
-- ORDER BY rank DESC;

-- Example 3: Weighted search
-- SELECT * FROM weighted_search('golang programming', 1);

-- Example 4: Find similar entries
-- SELECT * FROM find_similar_entries(123, 0.3, 5);

-- Example 5: Search specific field with highlighting
-- SELECT id, 
--        ts_headline('english', data->>'content', plainto_tsquery('english', 'golang')) as highlighted
-- FROM content_entries
-- WHERE to_tsvector('english', data->>'content') @@ plainto_tsquery('english', 'golang');

-- ==========================================
-- PERFORMANCE MONITORING
-- ==========================================

-- Create view for slow queries
CREATE OR REPLACE VIEW slow_searches AS
SELECT 
    query,
    AVG(execution_time_ms) as avg_time,
    MAX(execution_time_ms) as max_time,
    COUNT(*) as count
FROM search_logs
WHERE execution_time_ms > 1000 -- queries slower than 1 second
AND created_at > NOW() - INTERVAL '7 days'
GROUP BY query
ORDER BY avg_time DESC;

COMMENT ON TABLE search_logs IS 'Logs all search queries for analytics and optimization';
COMMENT ON COLUMN search_logs.execution_time_ms IS 'Query execution time in milliseconds';
COMMENT ON INDEX idx_content_entries_search_vector IS 'GIN index for fast full-text search';