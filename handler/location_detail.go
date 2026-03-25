package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"

	appErrors "github.com/howsworkingthere/hows-working-there-api/errors"
	"github.com/howsworkingthere/hows-working-there-api/models"
	"gofr.dev/pkg/gofr"
)

// GetLocationDetail handles GET /api/v1/locations/{location_id}/detail
// Returns comprehensive detail for a single location including aggregated stats.
// Uses a single CTE-based query to fetch all data in one database round-trip.
func GetLocationDetail(c *gofr.Context) (interface{}, error) {
	locationID := c.PathParam("location_id")
	if locationID == "" {
		return nil, appErrors.BadRequestError{Message: "location_id path parameter is required"}
	}

	query := fmt.Sprintf(`
	WITH loc AS (
		SELECT id, name, address, latitude, longitude, category, mapkit_poi_category, created_at, updated_at
		FROM locations WHERE id = '%s'
	),
	checkin_noise AS (
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
		WHERE ci.location_id = '%s'
	),
	isp_speeds AS (
		SELECT
			COALESCE(isp_name, 'Unknown') AS isp_name,
			ROUND(AVG(download_speed_mbps), 2) AS avg_download_mbps,
			ROUND(AVG(upload_speed_mbps), 2) AS avg_upload_mbps,
			ROUND(AVG(latency_ms), 0) AS avg_latency_ms,
			ROUND(AVG(jitter), 2) AS avg_jitter,
			COUNT(*) AS test_count
		FROM speed_tests
		WHERE location_id = '%s' AND skipped = FALSE
		GROUP BY isp_name
	),
	ws AS (
		SELECT
			COUNT(*) AS total_ratings,
			ROUND(100.0 * COUNT(*) FILTER (WHERE outlets_at_bar = TRUE)  / NULLIF(COUNT(*), 0), 1) AS pct_outlets_at_bar,
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
		WHERE location_id = '%s'
	)
	SELECT
		-- location
		loc.id, loc.name, loc.address, loc.latitude, loc.longitude,
		loc.category, loc.mapkit_poi_category, loc.created_at, loc.updated_at,
		-- check-in + noise
		cn.total_check_ins, cn.unique_users, cn.first_check_in, cn.last_check_in,
		cn.avg_decibels, cn.avg_peak_decibels, cn.noise_test_count,
		-- ISP speeds as JSON array
		COALESCE((
			SELECT json_agg(json_build_object(
				'isp_name', s.isp_name,
				'avg_download_mbps', s.avg_download_mbps,
				'avg_upload_mbps', s.avg_upload_mbps,
				'avg_latency_ms', s.avg_latency_ms,
				'avg_jitter', s.avg_jitter,
				'test_count', s.test_count
			) ORDER BY s.test_count DESC)
			FROM isp_speeds s
		), '[]'::json) AS speed_by_isp,
		-- workspace ratings
		ws.total_ratings, ws.pct_outlets_at_bar, ws.pct_outlets_at_table,
		ws.pct_crowdedness_empty, ws.pct_crowdedness_somewhat, ws.pct_crowdedness_crowded,
		ws.pct_ease_easy, ws.pct_ease_moderate, ws.pct_ease_difficult,
		ws.pct_work_solo, ws.pct_work_team
	FROM loc
	CROSS JOIN checkin_noise cn
	CROSS JOIN ws`, locationID, locationID, locationID, locationID)

	var resp models.LocationDetailResponse
	var ispJSON []byte

	err := c.SQL.QueryRowContext(c, query).Scan(
		// location
		&resp.Location.ID, &resp.Location.Name, &resp.Location.Address,
		&resp.Location.Latitude, &resp.Location.Longitude,
		&resp.Location.Category, &resp.Location.MapkitPOICategory,
		&resp.Location.CreatedAt, &resp.Location.UpdatedAt,
		// check-in + noise
		&resp.TotalCheckIns, &resp.UniqueUsers,
		&resp.FirstCheckIn, &resp.LastCheckIn,
		&resp.Noise.AvgDecibels, &resp.Noise.AvgPeakDecibels, &resp.Noise.TestCount,
		// ISP speeds JSON
		&ispJSON,
		// workspace ratings
		&resp.WorkspaceRating.TotalRatings,
		&resp.WorkspaceRating.PctOutletsAtBar, &resp.WorkspaceRating.PctOutletsAtTable,
		&resp.WorkspaceRating.Crowdedness.Empty, &resp.WorkspaceRating.Crowdedness.SomewhatCrowded, &resp.WorkspaceRating.Crowdedness.Crowded,
		&resp.WorkspaceRating.EaseOfWork.Easy, &resp.WorkspaceRating.EaseOfWork.Moderate, &resp.WorkspaceRating.EaseOfWork.Difficult,
		&resp.WorkspaceRating.BestWorkType.Solo, &resp.WorkspaceRating.BestWorkType.Team,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, appErrors.NotFoundError{Message: fmt.Sprintf("location not found: %s", locationID)}
		}
		return nil, fmt.Errorf("failed to query location detail: %w", err)
	}

	// Unmarshal ISP speed JSON array
	if err := json.Unmarshal(ispJSON, &resp.SpeedByISP); err != nil {
		return nil, fmt.Errorf("failed to parse ISP speed data: %w", err)
	}
	if resp.SpeedByISP == nil {
		resp.SpeedByISP = []models.ISPSpeedStats{}
	}

	return resp, nil
}
