-- Migration: Remove 'both' from best_work_type
-- Run this against the live database BEFORE deploying the updated API.

-- 1. Update any existing 'both' rows to 'team' (since 'both' is encapsulated by 'team')
UPDATE workspace_ratings SET best_work_type = 'team' WHERE best_work_type = 'both';

-- 2. Drop the old CHECK constraint and add the new one without 'both'
ALTER TABLE workspace_ratings DROP CONSTRAINT IF EXISTS chk_best_work_type;
ALTER TABLE workspace_ratings ADD CONSTRAINT chk_best_work_type CHECK (best_work_type IN ('solo', 'team'));
