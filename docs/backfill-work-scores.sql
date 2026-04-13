-- Backfill: Compute work_score and time_of_day for all existing check-ins.
-- Run AFTER migrate-add-work-score.sql.
-- This is a one-time migration script.

UPDATE check_ins ci
SET
    work_score = (
        -- 1. Network Speed (0-30 pts)
        CASE WHEN st.skipped = TRUE OR st.id IS NULL THEN 0
        ELSE
            -- Download sub-score (0-12)
            (CASE
                WHEN st.download_speed_mbps >= 100 THEN 12
                WHEN st.download_speed_mbps >= 50  THEN 10
                WHEN st.download_speed_mbps >= 25  THEN 8
                WHEN st.download_speed_mbps >= 10  THEN 5
                WHEN st.download_speed_mbps >= 5   THEN 3
                ELSE 1
            END)
            +
            -- Upload sub-score (0-10)
            (CASE
                WHEN st.upload_speed_mbps >= 50 THEN 10
                WHEN st.upload_speed_mbps >= 20 THEN 8
                WHEN st.upload_speed_mbps >= 10 THEN 6
                WHEN st.upload_speed_mbps >= 5  THEN 4
                ELSE 1
            END)
            +
            -- Jitter sub-score (0-8)
            (CASE
                WHEN st.jitter < 2   THEN 8
                WHEN st.jitter < 5   THEN 6
                WHEN st.jitter < 15  THEN 4
                WHEN st.jitter < 30  THEN 2
                ELSE 1
            END)
        END
    )
    +
    (
        -- 2. Noise / Decibels (0-20 pts)
        CASE WHEN nl.skipped = TRUE OR nl.id IS NULL THEN 0
        ELSE
            CASE
                WHEN nl.average_decibels <= 30 THEN 20
                WHEN nl.average_decibels <= 40 THEN 17
                WHEN nl.average_decibels <= 50 THEN 14
                WHEN nl.average_decibels <= 60 THEN 10
                WHEN nl.average_decibels <= 70 THEN 6
                WHEN nl.average_decibels <= 80 THEN 3
                ELSE 1
            END
        END
    )
    +
    (
        -- 3. Ease of Work (0-18 pts)
        CASE WHEN wr.id IS NULL THEN 0
        ELSE
            CASE wr.ease_of_work
                WHEN 1 THEN 18  -- Easy
                WHEN 2 THEN 10  -- Moderate
                WHEN 3 THEN 3   -- Difficult
                ELSE 0
            END
        END
    )
    +
    (
        -- 4. Outlet Availability (0-14 pts)
        CASE WHEN wr.id IS NULL THEN 0
        ELSE
            CASE
                WHEN wr.outlets_at_bar AND wr.outlets_at_table THEN 14
                WHEN wr.outlets_at_bar OR wr.outlets_at_table  THEN 8
                ELSE 2
            END
        END
    )
    +
    (
        -- 5. Crowdedness (0-12 pts)
        CASE WHEN wr.id IS NULL THEN 0
        ELSE
            CASE wr.crowdedness
                WHEN 1 THEN 12  -- Empty
                WHEN 2 THEN 7   -- Somewhat Crowded
                WHEN 3 THEN 2   -- Crowded
                ELSE 0
            END
        END
    )
    +
    (
        -- 6. Work Type (0-6 pts)
        CASE WHEN wr.id IS NULL THEN 0
        ELSE
            CASE wr.best_work_type
                WHEN 'team' THEN 6
                WHEN 'solo' THEN 3
                ELSE 0
            END
        END
    ),

    time_of_day = CASE
        WHEN EXTRACT(HOUR FROM ci.timestamp) >= 6  AND EXTRACT(HOUR FROM ci.timestamp) < 12 THEN 'morning'
        WHEN EXTRACT(HOUR FROM ci.timestamp) >= 12 AND EXTRACT(HOUR FROM ci.timestamp) < 18 THEN 'afternoon'
        ELSE 'evening'
    END

FROM check_ins ci2
LEFT JOIN speed_tests st ON st.check_in_id = ci2.id
LEFT JOIN noise_levels nl ON nl.check_in_id = ci2.id
LEFT JOIN workspace_ratings wr ON wr.check_in_id = ci2.id
WHERE ci.id = ci2.id
  AND ci.work_score IS NULL;
