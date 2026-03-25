# Location Detail API Endpoint

## Overview

A new endpoint that returns comprehensive detail for a single location, including base location data, aggregated check-in stats, noise averages, speed test averages grouped by ISP, and workspace rating distributions.

This replaces the need to rely on the flattened `v_location_averages` view for the location detail screen.

## Endpoint

```
GET /api/v1/locations/{location_id}/detail
```

- **Auth:** JWT required (same Auth0 middleware as all other endpoints)
- **Path Param:** `location_id` — UUID of the location
- **Response:** Single JSON object (wrapped in GoFr `data` envelope)
- **Errors:** 400 if missing location_id, 404 if location not found

## Response Shape

```json
{
  "location": {
    "id": "uuid",
    "name": "Blue Bottle Coffee",
    "address": "123 Main St, San Francisco, CA",
    "latitude": 37.7749,
    "longitude": -122.4194,
    "category": "cafe",
    "mapkit_poi_category": "MKPOICategory.cafe",
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-01-15T10:30:00Z"
  },
  "total_check_ins": 12,
  "unique_users": 5,
  "first_check_in": "2025-01-15T10:30:00Z",
  "last_check_in": "2025-03-18T14:22:00Z",
  "noise": {
    "avg_decibels": 52.3,
    "avg_peak_decibels": 68.1,
    "test_count": 10
  },
  "speed_by_isp": [
    {
      "isp_name": "Comcast",
      "avg_download_mbps": 45.20,
      "avg_upload_mbps": 12.10,
      "avg_latency_ms": 18,
      "avg_jitter": 3.20,
      "test_count": 7
    },
    {
      "isp_name": "AT&T",
      "avg_download_mbps": 32.50,
      "avg_upload_mbps": 8.40,
      "avg_latency_ms": 24,
      "avg_jitter": 5.10,
      "test_count": 3
    }
  ],
  "workspace_ratings": {
    "total_ratings": 10,
    "pct_outlets_at_bar": 60.0,
    "pct_outlets_at_table": 80.0,
    "crowdedness": {
      "empty": 30.0,
      "somewhat_crowded": 50.0,
      "crowded": 20.0
    },
    "ease_of_work": {
      "easy": 40.0,
      "moderate": 40.0,
      "difficult": 20.0
    },
    "best_work_type": {
      "solo": 60.0,
      "team": 40.0
    }
  }
}
```

## SQL Queries

Four separate targeted queries (not one mega-join):

### 1. Location Base Data
```sql
SELECT id, name, address, latitude, longitude, category, mapkit_poi_category, created_at, updated_at
FROM locations WHERE id = $1
```

### 2. Check-in + Noise Summary
```sql
SELECT
    COUNT(DISTINCT ci.id) AS total_check_ins,
    COUNT(DISTINCT ci.user_id) AS unique_users,
    MIN(ci.timestamp) AS first_check_in,
    MAX(ci.timestamp) AS last_check_in,
    ROUND(AVG(nl.average_decibels) FILTER (WHERE nl.skipped = FALSE), 1) AS avg_decibels,
    ROUND(AVG(nl.peak_decibels) FILTER (WHERE nl.skipped = FALSE), 1) AS avg_peak_decibels,
    COUNT(nl.id) FILTER (WHERE nl.skipped = FALSE) AS noise_test_count
FROM check_ins ci
LEFT JOIN noise_levels nl ON nl.check_in_id = ci.id
WHERE ci.location_id = $1
```

### 3. Speed by ISP
```sql
SELECT
    COALESCE(isp_name, 'Unknown') AS isp_name,
    ROUND(AVG(download_speed_mbps), 2) AS avg_download_mbps,
    ROUND(AVG(upload_speed_mbps), 2) AS avg_upload_mbps,
    ROUND(AVG(latency_ms), 0) AS avg_latency_ms,
    ROUND(AVG(jitter), 2) AS avg_jitter,
    COUNT(*) AS test_count
FROM speed_tests
WHERE location_id = $1 AND skipped = FALSE
GROUP BY isp_name
ORDER BY test_count DESC
```

### 4. Workspace Rating Distributions
```sql
SELECT
    COUNT(*) AS total_ratings,
    ROUND(100.0 * COUNT(*) FILTER (WHERE outlets_at_bar = TRUE) / NULLIF(COUNT(*), 0), 1) AS pct_outlets_at_bar,
    ROUND(100.0 * COUNT(*) FILTER (WHERE outlets_at_table = TRUE) / NULLIF(COUNT(*), 0), 1) AS pct_outlets_at_table,
    ROUND(100.0 * COUNT(*) FILTER (WHERE crowdedness = 1) / NULLIF(COUNT(*), 0), 1) AS pct_crowdedness_empty,
    ROUND(100.0 * COUNT(*) FILTER (WHERE crowdedness = 2) / NULLIF(COUNT(*), 0), 1) AS pct_crowdedness_somewhat,
    ROUND(100.0 * COUNT(*) FILTER (WHERE crowdedness = 3) / NULLIF(COUNT(*), 0), 1) AS pct_crowdedness_crowded,
    ROUND(100.0 * COUNT(*) FILTER (WHERE ease_of_work = 1) / NULLIF(COUNT(*), 0), 1) AS pct_ease_easy,
    ROUND(100.0 * COUNT(*) FILTER (WHERE ease_of_work = 2) / NULLIF(COUNT(*), 0), 1) AS pct_ease_moderate,
    ROUND(100.0 * COUNT(*) FILTER (WHERE ease_of_work = 3) / NULLIF(COUNT(*), 0), 1) AS pct_ease_difficult,
    ROUND(100.0 * COUNT(*) FILTER (WHERE best_work_type = 'solo') / NULLIF(COUNT(*), 0), 1) AS pct_work_solo,
    ROUND(100.0 * COUNT(*) FILTER (WHERE best_work_type = 'team') / NULLIF(COUNT(*), 0), 1) AS pct_work_team
FROM workspace_ratings
WHERE location_id = $1
```

## Tables Referenced

| Table | Columns Used |
|---|---|
| `locations` | All columns |
| `check_ins` | `id`, `user_id`, `location_id`, `timestamp` |
| `noise_levels` | `check_in_id`, `location_id`, `average_decibels`, `peak_decibels`, `skipped` |
| `speed_tests` | `location_id`, `isp_name`, `download_speed_mbps`, `upload_speed_mbps`, `latency_ms`, `jitter`, `skipped` |
| `workspace_ratings` | `location_id`, `outlets_at_bar`, `outlets_at_table`, `crowdedness`, `ease_of_work`, `best_work_type` |

## Existing Indexes Leveraged

- `idx_check_ins_location` on `check_ins(location_id)`
- `idx_speed_tests_location` on `speed_tests(location_id)`
- `idx_speed_tests_isp` on `speed_tests(isp_name)`
- `idx_noise_levels_location` on `noise_levels(location_id)`
- `idx_workspace_ratings_location` on `workspace_ratings(location_id)`
