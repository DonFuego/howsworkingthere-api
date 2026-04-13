-- Migration: Add work_score and time_of_day columns to check_ins
-- Run this against the production database before deploying the new API version.

-- ============================================
-- Step 1: Add columns to check_ins
-- ============================================
ALTER TABLE check_ins ADD COLUMN IF NOT EXISTS work_score SMALLINT;
ALTER TABLE check_ins ADD COLUMN IF NOT EXISTS time_of_day VARCHAR(10);

-- Constraints
ALTER TABLE check_ins ADD CONSTRAINT chk_work_score
    CHECK (work_score >= 0 AND work_score <= 100);
ALTER TABLE check_ins ADD CONSTRAINT chk_time_of_day
    CHECK (time_of_day IN ('morning', 'afternoon', 'evening'));

-- Index for time-of-day queries
CREATE INDEX IF NOT EXISTS idx_check_ins_work_score ON check_ins(work_score);
CREATE INDEX IF NOT EXISTS idx_check_ins_time_of_day ON check_ins(time_of_day);

-- ============================================
-- Step 2: Update v_location_averages view
-- ============================================
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
    MAX(ci.timestamp) AS last_check_in,

    -- Work Score aggregates
    ROUND(AVG(ci.work_score), 0) AS avg_work_score,
    ROUND(AVG(ci.work_score) FILTER (WHERE ci.time_of_day = 'morning'), 0) AS avg_score_morning,
    ROUND(AVG(ci.work_score) FILTER (WHERE ci.time_of_day = 'afternoon'), 0) AS avg_score_afternoon,
    ROUND(AVG(ci.work_score) FILTER (WHERE ci.time_of_day = 'evening'), 0) AS avg_score_evening

FROM locations loc
LEFT JOIN check_ins ci ON ci.location_id = loc.id
LEFT JOIN speed_tests st ON st.check_in_id = ci.id
LEFT JOIN noise_levels nl ON nl.check_in_id = ci.id
LEFT JOIN workspace_ratings wr ON wr.check_in_id = ci.id
GROUP BY loc.id, loc.name, loc.address, loc.latitude, loc.longitude, loc.category;

-- ============================================
-- Step 3: Update v_user_location_averages view
-- ============================================
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
    MAX(ci.timestamp) AS my_last_check_in,

    -- User's Work Score aggregates at this location
    ROUND(AVG(ci.work_score), 0) AS my_avg_work_score,
    ROUND(AVG(ci.work_score) FILTER (WHERE ci.time_of_day = 'morning'), 0) AS my_avg_score_morning,
    ROUND(AVG(ci.work_score) FILTER (WHERE ci.time_of_day = 'afternoon'), 0) AS my_avg_score_afternoon,
    ROUND(AVG(ci.work_score) FILTER (WHERE ci.time_of_day = 'evening'), 0) AS my_avg_score_evening

FROM check_ins ci
INNER JOIN locations loc ON ci.location_id = loc.id
LEFT JOIN speed_tests st ON st.check_in_id = ci.id
LEFT JOIN noise_levels nl ON nl.check_in_id = ci.id
LEFT JOIN workspace_ratings wr ON wr.check_in_id = ci.id
GROUP BY ci.user_id, loc.id, loc.name, loc.address, loc.latitude, loc.longitude, loc.category;

-- ============================================
-- Step 4: Update v_user_check_in_history view
-- ============================================
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
    wr.best_work_type,

    -- Work Score (this visit)
    ci.work_score,
    ci.time_of_day

FROM check_ins ci
INNER JOIN locations loc ON ci.location_id = loc.id
LEFT JOIN speed_tests st ON st.check_in_id = ci.id
LEFT JOIN noise_levels nl ON nl.check_in_id = ci.id
LEFT JOIN workspace_ratings wr ON wr.check_in_id = ci.id;

COMMENT ON COLUMN check_ins.work_score IS '0-100 composite work quality score computed at check-in time';
COMMENT ON COLUMN check_ins.time_of_day IS 'morning (6-12), afternoon (12-18), evening (18-6) — derived from check-in timestamp';
