-- Migration: Update v_location_averages and v_user_location_averages
-- 1. Filter legacy 'both' from most_common_work_type MODE()
-- 2. Add most_common_ease_of_work / my_most_common_ease_of_work columns

-- ============================================================
-- v_location_averages
-- ============================================================
DROP VIEW IF EXISTS v_location_averages CASCADE;
CREATE VIEW v_location_averages AS
SELECT
    loc.id AS location_id,
    loc.name AS location_name,
    loc.address AS location_address,
    loc.latitude,
    loc.longitude,
    loc.category AS location_category,

    COUNT(DISTINCT ci.id) AS total_check_ins,
    COUNT(DISTINCT ci.user_id) AS unique_users,

    ROUND(AVG(st.download_speed_mbps) FILTER (WHERE st.skipped = FALSE), 2) AS avg_download_mbps,
    ROUND(AVG(st.upload_speed_mbps) FILTER (WHERE st.skipped = FALSE), 2) AS avg_upload_mbps,
    ROUND(AVG(st.latency_ms) FILTER (WHERE st.skipped = FALSE), 0) AS avg_latency_ms,
    ROUND(AVG(st.jitter) FILTER (WHERE st.skipped = FALSE), 2) AS avg_jitter,
    COUNT(st.id) FILTER (WHERE st.skipped = FALSE) AS speed_test_count,

    ROUND(AVG(nl.average_decibels) FILTER (WHERE nl.skipped = FALSE), 1) AS avg_decibels,
    ROUND(AVG(nl.peak_decibels) FILTER (WHERE nl.skipped = FALSE), 1) AS avg_peak_decibels,
    COUNT(nl.id) FILTER (WHERE nl.skipped = FALSE) AS noise_test_count,

    ROUND(AVG(wr.crowdedness), 1) AS avg_crowdedness,
    ROUND(AVG(wr.ease_of_work), 1) AS avg_ease_of_work,
    COUNT(wr.id) AS rating_count,

    ROUND(100.0 * COUNT(wr.id) FILTER (WHERE wr.outlets_at_bar = TRUE) / NULLIF(COUNT(wr.id), 0), 0) AS pct_outlets_at_bar,
    ROUND(100.0 * COUNT(wr.id) FILTER (WHERE wr.outlets_at_table = TRUE) / NULLIF(COUNT(wr.id), 0), 0) AS pct_outlets_at_table,

    MODE() WITHIN GROUP (ORDER BY wr.best_work_type) FILTER (WHERE wr.best_work_type IN ('solo', 'team')) AS most_common_work_type,
    MODE() WITHIN GROUP (ORDER BY wr.ease_of_work) AS most_common_ease_of_work,

    MIN(ci.timestamp) AS first_check_in,
    MAX(ci.timestamp) AS last_check_in

FROM locations loc
LEFT JOIN check_ins ci ON ci.location_id = loc.id
LEFT JOIN speed_tests st ON st.check_in_id = ci.id
LEFT JOIN noise_levels nl ON nl.check_in_id = ci.id
LEFT JOIN workspace_ratings wr ON wr.check_in_id = ci.id
GROUP BY loc.id, loc.name, loc.address, loc.latitude, loc.longitude, loc.category;

-- ============================================================
-- v_user_location_averages
-- ============================================================
DROP VIEW IF EXISTS v_user_location_averages CASCADE;
CREATE VIEW v_user_location_averages AS
SELECT
    ci.user_id,

    loc.id AS location_id,
    loc.name AS location_name,
    loc.address AS location_address,
    loc.latitude,
    loc.longitude,
    loc.category AS location_category,

    COUNT(DISTINCT ci.id) AS my_check_ins,

    ROUND(AVG(st.download_speed_mbps) FILTER (WHERE st.skipped = FALSE), 2) AS my_avg_download_mbps,
    ROUND(AVG(st.upload_speed_mbps) FILTER (WHERE st.skipped = FALSE), 2) AS my_avg_upload_mbps,
    ROUND(AVG(st.latency_ms) FILTER (WHERE st.skipped = FALSE), 0) AS my_avg_latency_ms,
    ROUND(AVG(st.jitter) FILTER (WHERE st.skipped = FALSE), 2) AS my_avg_jitter,
    COUNT(st.id) FILTER (WHERE st.skipped = FALSE) AS my_speed_test_count,

    ROUND(AVG(nl.average_decibels) FILTER (WHERE nl.skipped = FALSE), 1) AS my_avg_decibels,
    ROUND(AVG(nl.peak_decibels) FILTER (WHERE nl.skipped = FALSE), 1) AS my_avg_peak_decibels,
    COUNT(nl.id) FILTER (WHERE nl.skipped = FALSE) AS my_noise_test_count,

    ROUND(AVG(wr.crowdedness), 1) AS my_avg_crowdedness,
    ROUND(AVG(wr.ease_of_work), 1) AS my_avg_ease_of_work,
    COUNT(wr.id) AS my_rating_count,

    ROUND(100.0 * COUNT(wr.id) FILTER (WHERE wr.outlets_at_bar = TRUE) / NULLIF(COUNT(wr.id), 0), 0) AS my_pct_outlets_at_bar,
    ROUND(100.0 * COUNT(wr.id) FILTER (WHERE wr.outlets_at_table = TRUE) / NULLIF(COUNT(wr.id), 0), 0) AS my_pct_outlets_at_table,

    MODE() WITHIN GROUP (ORDER BY wr.best_work_type) FILTER (WHERE wr.best_work_type IN ('solo', 'team')) AS my_most_common_work_type,
    MODE() WITHIN GROUP (ORDER BY wr.ease_of_work) AS my_most_common_ease_of_work,

    MIN(ci.timestamp) AS my_first_check_in,
    MAX(ci.timestamp) AS my_last_check_in

FROM check_ins ci
INNER JOIN locations loc ON ci.location_id = loc.id
LEFT JOIN speed_tests st ON st.check_in_id = ci.id
LEFT JOIN noise_levels nl ON nl.check_in_id = ci.id
LEFT JOIN workspace_ratings wr ON wr.check_in_id = ci.id
GROUP BY ci.user_id, loc.id, loc.name, loc.address, loc.latitude, loc.longitude, loc.category;
