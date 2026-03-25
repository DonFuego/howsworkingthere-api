-- Check-In Database Schema
-- Based on docs/check-in-api-model.md
-- PostgreSQL 12+ compatible

-- Set schema (adjust if needed)
SET search_path TO public;

-- Enable UUID extension (requires superuser or rds_superuser on AWS RDS)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================
-- Trigger Function: Auto-update updated_at
-- ============================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- ============================================
-- Table: locations
-- Stores unique place information
-- ============================================
CREATE TABLE IF NOT EXISTS locations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(500) NOT NULL,
    address VARCHAR(500) NOT NULL,
    latitude DECIMAL(10, 8) NOT NULL,
    longitude DECIMAL(11, 8) NOT NULL,

    -- Category (user-confirmed, mapped from MapKit POI category)
    category VARCHAR(20) NOT NULL DEFAULT 'other',
    -- Raw MapKit pointOfInterestCategory value for reference
    mapkit_poi_category VARCHAR(100),

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT chk_latitude CHECK (latitude >= -90 AND latitude <= 90),
    CONSTRAINT chk_longitude CHECK (longitude >= -180 AND longitude <= 180),
    CONSTRAINT chk_category CHECK (category IN (
        'cafe', 'restaurant_bar', 'hotel', 'park', 'library', 'office', 'coworking', 'other'
    )),

    -- Enforce one row per physical place
    CONSTRAINT uq_locations_name_coords UNIQUE (name, latitude, longitude)
);

-- Trigger for auto-updating updated_at
DROP TRIGGER IF EXISTS update_locations_updated_at ON locations;
CREATE TRIGGER update_locations_updated_at
    BEFORE UPDATE ON locations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Index for geospatial queries
-- Note: Use CREATE INDEX CONCURRENTLY in production to avoid table locks
CREATE INDEX IF NOT EXISTS idx_locations_coords ON locations(latitude, longitude);
CREATE INDEX IF NOT EXISTS idx_locations_name ON locations(name);
CREATE INDEX IF NOT EXISTS idx_locations_address ON locations(address);
CREATE INDEX IF NOT EXISTS idx_locations_category ON locations(category);

-- ============================================
-- Table: check_ins
-- Parent table representing a check-in event
-- ============================================
CREATE TABLE IF NOT EXISTS check_ins (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    location_id UUID NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign Key
    CONSTRAINT fk_check_ins_location
        FOREIGN KEY (location_id)
        REFERENCES locations(id)
        ON DELETE CASCADE,

    -- Ensure one check-in per user per location per timestamp
    CONSTRAINT uq_check_ins_user_location_time
        UNIQUE (user_id, location_id, timestamp)
);

-- Indexes for check_ins
CREATE INDEX IF NOT EXISTS idx_check_ins_user ON check_ins(user_id);
CREATE INDEX IF NOT EXISTS idx_check_ins_location ON check_ins(location_id);
CREATE INDEX IF NOT EXISTS idx_check_ins_timestamp ON check_ins(timestamp);

-- ============================================
-- Table: speed_tests
-- Network speed measurements for a check-in
-- ============================================
CREATE TABLE IF NOT EXISTS speed_tests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    check_in_id UUID NOT NULL,
    location_id UUID NOT NULL,

    -- Speed metrics
    download_speed_mbps DECIMAL(10, 2) NOT NULL,
    upload_speed_mbps DECIMAL(10, 2) NOT NULL,
    latency_ms INTEGER NOT NULL,
    jitter DECIMAL(8, 4) NOT NULL,

    -- Network details
    isp_name VARCHAR(255),
    -- INET is a PostgreSQL-native type for IP addresses with built-in validation
    ip_address INET,
    network_type VARCHAR(20) NOT NULL DEFAULT 'unknown',
    packet_loss_percent DECIMAL(5, 2),
    time_to_first_byte_ms INTEGER NOT NULL,

    -- Transfer stats
    download_transferred_mb DECIMAL(10, 2) NOT NULL,
    upload_transferred_mb DECIMAL(10, 2) NOT NULL,

    -- Server info
    server_domain VARCHAR(255),
    server_city VARCHAR(255),
    server_country VARCHAR(255),

    -- External reference
    speed_test_id VARCHAR(255) NOT NULL,

    -- Flags
    skipped BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign Keys
    CONSTRAINT fk_speed_tests_check_in
        FOREIGN KEY (check_in_id)
        REFERENCES check_ins(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_speed_tests_location
        FOREIGN KEY (location_id)
        REFERENCES locations(id)
        ON DELETE CASCADE,

    -- Constraints
    CONSTRAINT chk_network_type
        CHECK (network_type IN ('wifi', 'cellular', 'unknown')),

    CONSTRAINT chk_download_speed
        CHECK (download_speed_mbps >= 0),

    CONSTRAINT chk_upload_speed
        CHECK (upload_speed_mbps >= 0),

    CONSTRAINT chk_latency
        CHECK (latency_ms >= 0),

    CONSTRAINT chk_jitter
        CHECK (jitter >= 0),

    CONSTRAINT chk_packet_loss
        CHECK (packet_loss_percent >= 0 AND packet_loss_percent <= 100),

    CONSTRAINT chk_ttfb
        CHECK (time_to_first_byte_ms >= 0)
);

-- Indexes for speed_tests
CREATE INDEX IF NOT EXISTS idx_speed_tests_check_in ON speed_tests(check_in_id);
CREATE INDEX IF NOT EXISTS idx_speed_tests_location ON speed_tests(location_id);
CREATE INDEX IF NOT EXISTS idx_speed_tests_isp ON speed_tests(isp_name);
CREATE INDEX IF NOT EXISTS idx_speed_tests_network_type ON speed_tests(network_type);

-- ============================================
-- Table: noise_levels
-- Ambient noise measurements for a check-in
-- ============================================
CREATE TABLE IF NOT EXISTS noise_levels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    check_in_id UUID NOT NULL,
    location_id UUID NOT NULL,

    -- Decibel measurements
    average_decibels DECIMAL(5, 2) NOT NULL,
    peak_decibels DECIMAL(5, 2) NOT NULL,
    duration_seconds DECIMAL(5, 2) NOT NULL,

    -- Flags
    skipped BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign Keys
    CONSTRAINT fk_noise_levels_check_in
        FOREIGN KEY (check_in_id)
        REFERENCES check_ins(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_noise_levels_location
        FOREIGN KEY (location_id)
        REFERENCES locations(id)
        ON DELETE CASCADE,

    -- Constraints
    CONSTRAINT chk_average_decibels
        CHECK (average_decibels >= 0 AND average_decibels <= 200),

    CONSTRAINT chk_peak_decibels
        CHECK (peak_decibels >= 0 AND peak_decibels <= 200),

    CONSTRAINT chk_duration
        CHECK (duration_seconds > 0 AND duration_seconds <= 300),

    CONSTRAINT chk_peak_gte_average
        CHECK (peak_decibels >= average_decibels)
);

-- Indexes for noise_levels
CREATE INDEX IF NOT EXISTS idx_noise_levels_check_in ON noise_levels(check_in_id);
CREATE INDEX IF NOT EXISTS idx_noise_levels_location ON noise_levels(location_id);

-- ============================================
-- Table: workspace_ratings
-- User-submitted workspace ratings for a check-in
-- ============================================
CREATE TABLE IF NOT EXISTS workspace_ratings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    check_in_id UUID NOT NULL,
    location_id UUID NOT NULL,

    -- Power outlets
    outlets_at_bar BOOLEAN NOT NULL,
    outlets_at_table BOOLEAN NOT NULL,

    -- Ratings (enums as integers)
    crowdedness INTEGER NOT NULL,
    ease_of_work INTEGER NOT NULL,
    best_work_type VARCHAR(20) NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Foreign Keys
    CONSTRAINT fk_workspace_ratings_check_in
        FOREIGN KEY (check_in_id)
        REFERENCES check_ins(id)
        ON DELETE CASCADE,

    CONSTRAINT fk_workspace_ratings_location
        FOREIGN KEY (location_id)
        REFERENCES locations(id)
        ON DELETE CASCADE,

    -- Constraints
    CONSTRAINT chk_crowdedness
        CHECK (crowdedness IN (1, 2, 3)),

    CONSTRAINT chk_ease_of_work
        CHECK (ease_of_work IN (1, 2, 3)),

    CONSTRAINT chk_best_work_type
        CHECK (best_work_type IN ('solo', 'team'))
);

-- Indexes for workspace_ratings
CREATE INDEX IF NOT EXISTS idx_workspace_ratings_check_in ON workspace_ratings(check_in_id);
CREATE INDEX IF NOT EXISTS idx_workspace_ratings_location ON workspace_ratings(location_id);
CREATE INDEX IF NOT EXISTS idx_workspace_ratings_crowdedness ON workspace_ratings(crowdedness);
CREATE INDEX IF NOT EXISTS idx_workspace_ratings_ease ON workspace_ratings(ease_of_work);
CREATE INDEX IF NOT EXISTS idx_workspace_ratings_work_type ON workspace_ratings(best_work_type);

-- ============================================
-- Views for Common Queries
-- ============================================

-- View: Single check-in with all details (one row per check-in)
CREATE OR REPLACE VIEW v_check_in_details AS
SELECT
    ci.id AS check_in_id,
    ci.user_id,
    ci.timestamp AS check_in_timestamp,
    ci.created_at AS check_in_created_at,

    -- Location
    loc.id AS location_id,
    loc.name AS location_name,
    loc.address AS location_address,
    loc.latitude,
    loc.longitude,
    loc.category AS location_category,

    -- Speed Test
    st.id AS speed_test_id,
    st.download_speed_mbps,
    st.upload_speed_mbps,
    st.latency_ms,
    st.jitter,
    st.isp_name,
    st.ip_address,
    st.network_type,
    st.packet_loss_percent,
    st.time_to_first_byte_ms,
    st.server_city,
    st.server_country,
    st.speed_test_id AS external_speed_test_id,
    st.skipped AS speed_test_skipped,

    -- Noise Level
    nl.id AS noise_level_id,
    nl.average_decibels,
    nl.peak_decibels,
    nl.duration_seconds,
    nl.skipped AS noise_level_skipped,

    -- Workspace Ratings
    wr.id AS workspace_rating_id,
    wr.outlets_at_bar,
    wr.outlets_at_table,
    wr.crowdedness,
    wr.ease_of_work,
    wr.best_work_type

FROM check_ins ci
INNER JOIN locations loc ON ci.location_id = loc.id
LEFT JOIN speed_tests st ON st.check_in_id = ci.id
LEFT JOIN noise_levels nl ON nl.check_in_id = ci.id
LEFT JOIN workspace_ratings wr ON wr.check_in_id = ci.id;

-- View: Averaged scores per location across all check-ins
-- This is the primary view for displaying location quality over time
CREATE OR REPLACE VIEW v_location_averages AS
SELECT
    loc.id AS location_id,
    loc.name AS location_name,
    loc.address AS location_address,
    loc.latitude,
    loc.longitude,
    loc.category AS location_category,

    -- Check-in counts
    COUNT(DISTINCT ci.id) AS total_check_ins,
    COUNT(DISTINCT ci.user_id) AS unique_users,

    -- Speed test averages (excludes skipped tests)
    ROUND(AVG(st.download_speed_mbps) FILTER (WHERE st.skipped = FALSE), 2) AS avg_download_mbps,
    ROUND(AVG(st.upload_speed_mbps) FILTER (WHERE st.skipped = FALSE), 2) AS avg_upload_mbps,
    ROUND(AVG(st.latency_ms) FILTER (WHERE st.skipped = FALSE), 0) AS avg_latency_ms,
    ROUND(AVG(st.jitter) FILTER (WHERE st.skipped = FALSE), 2) AS avg_jitter,
    COUNT(st.id) FILTER (WHERE st.skipped = FALSE) AS speed_test_count,

    -- Noise level averages (excludes skipped tests)
    ROUND(AVG(nl.average_decibels) FILTER (WHERE nl.skipped = FALSE), 1) AS avg_decibels,
    ROUND(AVG(nl.peak_decibels) FILTER (WHERE nl.skipped = FALSE), 1) AS avg_peak_decibels,
    COUNT(nl.id) FILTER (WHERE nl.skipped = FALSE) AS noise_test_count,

    -- Workspace rating averages
    ROUND(AVG(wr.crowdedness), 1) AS avg_crowdedness,
    ROUND(AVG(wr.ease_of_work), 1) AS avg_ease_of_work,
    COUNT(wr.id) AS rating_count,

    -- Outlet availability (percentage of check-ins reporting outlets)
    ROUND(100.0 * COUNT(wr.id) FILTER (WHERE wr.outlets_at_bar = TRUE) / NULLIF(COUNT(wr.id), 0), 0) AS pct_outlets_at_bar,
    ROUND(100.0 * COUNT(wr.id) FILTER (WHERE wr.outlets_at_table = TRUE) / NULLIF(COUNT(wr.id), 0), 0) AS pct_outlets_at_table,

    -- Most common work type (mode, excluding legacy 'both')
    MODE() WITHIN GROUP (ORDER BY wr.best_work_type) FILTER (WHERE wr.best_work_type IN ('solo', 'team')) AS most_common_work_type,

    -- Most common ease of work (mode: 1=easy, 2=moderate, 3=difficult)
    MODE() WITHIN GROUP (ORDER BY wr.ease_of_work) AS most_common_ease_of_work,

    -- Activity window
    MIN(ci.timestamp) AS first_check_in,
    MAX(ci.timestamp) AS last_check_in

FROM locations loc
LEFT JOIN check_ins ci ON ci.location_id = loc.id
LEFT JOIN speed_tests st ON st.check_in_id = ci.id
LEFT JOIN noise_levels nl ON nl.check_in_id = ci.id
LEFT JOIN workspace_ratings wr ON wr.check_in_id = ci.id
GROUP BY loc.id, loc.name, loc.address, loc.latitude, loc.longitude, loc.category;

-- View: A single user's averaged scores per location (only locations they've visited)
-- Usage: SELECT * FROM v_user_location_averages WHERE user_id = 'auth0|abc123';
CREATE OR REPLACE VIEW v_user_location_averages AS
SELECT
    ci.user_id,

    -- Location
    loc.id AS location_id,
    loc.name AS location_name,
    loc.address AS location_address,
    loc.latitude,
    loc.longitude,
    loc.category AS location_category,

    -- User's check-in counts at this location
    COUNT(DISTINCT ci.id) AS my_check_ins,

    -- User's speed test averages at this location
    ROUND(AVG(st.download_speed_mbps) FILTER (WHERE st.skipped = FALSE), 2) AS my_avg_download_mbps,
    ROUND(AVG(st.upload_speed_mbps) FILTER (WHERE st.skipped = FALSE), 2) AS my_avg_upload_mbps,
    ROUND(AVG(st.latency_ms) FILTER (WHERE st.skipped = FALSE), 0) AS my_avg_latency_ms,
    ROUND(AVG(st.jitter) FILTER (WHERE st.skipped = FALSE), 2) AS my_avg_jitter,
    COUNT(st.id) FILTER (WHERE st.skipped = FALSE) AS my_speed_test_count,

    -- User's noise level averages at this location
    ROUND(AVG(nl.average_decibels) FILTER (WHERE nl.skipped = FALSE), 1) AS my_avg_decibels,
    ROUND(AVG(nl.peak_decibels) FILTER (WHERE nl.skipped = FALSE), 1) AS my_avg_peak_decibels,
    COUNT(nl.id) FILTER (WHERE nl.skipped = FALSE) AS my_noise_test_count,

    -- User's workspace rating averages at this location
    ROUND(AVG(wr.crowdedness), 1) AS my_avg_crowdedness,
    ROUND(AVG(wr.ease_of_work), 1) AS my_avg_ease_of_work,
    COUNT(wr.id) AS my_rating_count,

    -- User's outlet reports at this location
    ROUND(100.0 * COUNT(wr.id) FILTER (WHERE wr.outlets_at_bar = TRUE) / NULLIF(COUNT(wr.id), 0), 0) AS my_pct_outlets_at_bar,
    ROUND(100.0 * COUNT(wr.id) FILTER (WHERE wr.outlets_at_table = TRUE) / NULLIF(COUNT(wr.id), 0), 0) AS my_pct_outlets_at_table,

    -- User's most common work type at this location (excluding legacy 'both')
    MODE() WITHIN GROUP (ORDER BY wr.best_work_type) FILTER (WHERE wr.best_work_type IN ('solo', 'team')) AS my_most_common_work_type,

    -- User's most common ease of work at this location (mode: 1=easy, 2=moderate, 3=difficult)
    MODE() WITHIN GROUP (ORDER BY wr.ease_of_work) AS my_most_common_ease_of_work,

    -- User's activity window at this location
    MIN(ci.timestamp) AS my_first_check_in,
    MAX(ci.timestamp) AS my_last_check_in

FROM check_ins ci
INNER JOIN locations loc ON ci.location_id = loc.id
LEFT JOIN speed_tests st ON st.check_in_id = ci.id
LEFT JOIN noise_levels nl ON nl.check_in_id = ci.id
LEFT JOIN workspace_ratings wr ON wr.check_in_id = ci.id
GROUP BY ci.user_id, loc.id, loc.name, loc.address, loc.latitude, loc.longitude, loc.category;

-- View: A single user's check-in history with full details (chronological)
-- Usage: SELECT * FROM v_user_check_in_history WHERE user_id = 'auth0|abc123' ORDER BY check_in_timestamp DESC;
CREATE OR REPLACE VIEW v_user_check_in_history AS
SELECT
    ci.user_id,
    ci.id AS check_in_id,
    ci.timestamp AS check_in_timestamp,

    -- Location
    loc.id AS location_id,
    loc.name AS location_name,
    loc.address AS location_address,
    loc.latitude,
    loc.longitude,
    loc.category AS location_category,

    -- Speed Test (this visit)
    st.download_speed_mbps,
    st.upload_speed_mbps,
    st.latency_ms,
    st.jitter,
    st.isp_name,
    st.network_type,
    st.skipped AS speed_test_skipped,

    -- Noise Level (this visit)
    nl.average_decibels,
    nl.peak_decibels,
    nl.skipped AS noise_level_skipped,

    -- Workspace Ratings (this visit)
    wr.outlets_at_bar,
    wr.outlets_at_table,
    wr.crowdedness,
    wr.ease_of_work,
    wr.best_work_type

FROM check_ins ci
INNER JOIN locations loc ON ci.location_id = loc.id
LEFT JOIN speed_tests st ON st.check_in_id = ci.id
LEFT JOIN noise_levels nl ON nl.check_in_id = ci.id
LEFT JOIN workspace_ratings wr ON wr.check_in_id = ci.id;

-- ============================================
-- Comments for Documentation
-- ============================================

-- Relationships:
--   locations   1 ──< many  check_ins       (one location, many check-ins by different users over time)
--   check_ins   1 ──< many  speed_tests     (one check-in can have one speed test, but a location accumulates many)
--   check_ins   1 ──< many  noise_levels    (one check-in can have one noise test, but a location accumulates many)
--   check_ins   1 ──< many  workspace_ratings (one check-in can have one rating, but a location accumulates many)
--
-- Views:
--   v_check_in_details       — Raw detail for a single check-in (all tables joined)
--   v_location_averages      — Global averaged scores per location across ALL users
--   v_user_location_averages — A user's own averaged scores per location (only their visits)
--   v_user_check_in_history  — A user's chronological check-in history with full details

COMMENT ON TABLE locations IS 'Unique physical places — each location exists exactly once (enforced by uq_locations_name_coords)';
COMMENT ON TABLE check_ins IS 'A single user visit to a location — many check-ins reference one location';
COMMENT ON TABLE speed_tests IS 'Network speed test results — many tests per location over time, averaged via v_location_averages';
COMMENT ON TABLE noise_levels IS 'Ambient noise measurements — many tests per location over time, averaged via v_location_averages';
COMMENT ON TABLE workspace_ratings IS 'User-submitted workspace ratings — many ratings per location over time, averaged via v_location_averages';

COMMENT ON COLUMN workspace_ratings.crowdedness IS '1=Empty, 2=Somewhat Crowded, 3=Crowded';
COMMENT ON COLUMN workspace_ratings.ease_of_work IS '1=Easy, 2=Moderate, 3=Difficult';
COMMENT ON COLUMN workspace_ratings.best_work_type IS 'solo or team';
