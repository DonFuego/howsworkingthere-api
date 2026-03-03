package handler

import (
	"fmt"
	"strconv"
	"strings"

	appErrors "github.com/howsworkingthere/hows-working-there-api/errors"
	"github.com/howsworkingthere/hows-working-there-api/middleware"
	"github.com/howsworkingthere/hows-working-there-api/models"
	"gofr.dev/pkg/gofr"
)

// GetUserLocations handles GET /api/v1/users/{user_id}/locations
// Returns all locations tested by a specific user with averaged scores from v_user_location_averages.
func GetUserLocations(c *gofr.Context) (interface{}, error) {
	userID := c.PathParam("user_id")
	if userID == "" {
		return nil, fmt.Errorf("user_id path parameter is required")
	}

	// Verify the authenticated user matches the requested user_id
	authedUserID, ok := middleware.GetUserIDFromContext(c.Request.Context())
	if !ok {
		return nil, fmt.Errorf("unable to determine authenticated user")
	}
	if userID != authedUserID {
		return nil, appErrors.ForbiddenError{Message: "access denied: you can only view your own locations"}
	}

	rows, err := c.SQL.QueryContext(c,
		`SELECT
			user_id, location_id, location_name, location_address,
			latitude, longitude, location_category,
			my_check_ins,
			my_avg_download_mbps, my_avg_upload_mbps, my_avg_latency_ms, my_avg_jitter,
			my_speed_test_count,
			my_avg_decibels, my_avg_peak_decibels, my_noise_test_count,
			my_avg_crowdedness, my_avg_ease_of_work, my_rating_count,
			my_pct_outlets_at_bar, my_pct_outlets_at_table,
			my_most_common_work_type,
			my_first_check_in, my_last_check_in
		FROM v_user_location_averages
		WHERE user_id = $1
		ORDER BY my_last_check_in DESC`, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query user locations: %w", err)
	}
	defer rows.Close()

	var results []models.UserLocationAverage
	for rows.Next() {
		var r models.UserLocationAverage
		if err := rows.Scan(
			&r.UserID, &r.LocationID, &r.LocationName, &r.LocationAddress,
			&r.Latitude, &r.Longitude, &r.LocationCategory,
			&r.MyCheckIns,
			&r.MyAvgDownloadMbps, &r.MyAvgUploadMbps, &r.MyAvgLatencyMs, &r.MyAvgJitter,
			&r.MySpeedTestCount,
			&r.MyAvgDecibels, &r.MyAvgPeakDecibels, &r.MyNoiseTestCount,
			&r.MyAvgCrowdedness, &r.MyAvgEaseOfWork, &r.MyRatingCount,
			&r.MyPctOutletsAtBar, &r.MyPctOutletsAtTable,
			&r.MyMostCommonWorkType,
			&r.MyFirstCheckIn, &r.MyLastCheckIn,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user location: %w", err)
		}
		results = append(results, r)
	}

	if results == nil {
		results = []models.UserLocationAverage{}
	}

	return results, nil
}

// GetAllLocations handles GET /api/v1/locations
// Returns all tested locations with averaged scores from v_location_averages.
// Supports optional filtering by category, geo-bounding, and pagination.
func GetAllLocations(c *gofr.Context) (interface{}, error) {
	category := strings.TrimSpace(c.Param("category"))
	latStr := c.Param("latitude")
	lngStr := c.Param("longitude")
	radiusStr := c.Param("radius_km")
	limitStr := c.Param("limit")
	offsetStr := c.Param("offset")

	var conditions []string
	var args []interface{}
	argIdx := 1

	if category != "" {
		conditions = append(conditions, fmt.Sprintf("location_category = $%d", argIdx))
		args = append(args, category)
		argIdx++
	}

	if latStr != "" && lngStr != "" {
		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid latitude value")
		}
		lng, err := strconv.ParseFloat(lngStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid longitude value")
		}

		radiusKm := 10.0
		if radiusStr != "" {
			r, err := strconv.ParseFloat(radiusStr, 64)
			if err == nil && r > 0 {
				radiusKm = r
			}
		}

		haversine := fmt.Sprintf(
			`(6371 * acos(
				LEAST(1.0, cos(radians($%d)) * cos(radians(latitude)) *
				cos(radians(longitude) - radians($%d)) +
				sin(radians($%d)) * sin(radians(latitude)))
			)) <= $%d`,
			argIdx, argIdx+1, argIdx+2, argIdx+3,
		)
		conditions = append(conditions, haversine)
		args = append(args, lat, lng, lat, radiusKm)
		argIdx += 4
	}

	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}

	offset := 0
	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	query := `SELECT
		location_id, location_name, location_address,
		latitude, longitude, location_category,
		total_check_ins, unique_users,
		avg_download_mbps, avg_upload_mbps, avg_latency_ms, avg_jitter,
		speed_test_count,
		avg_decibels, avg_peak_decibels, noise_test_count,
		avg_crowdedness, avg_ease_of_work, rating_count,
		pct_outlets_at_bar, pct_outlets_at_table,
		most_common_work_type,
		first_check_in, last_check_in
	FROM v_location_averages`

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY total_check_ins DESC"
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := c.SQL.QueryContext(c, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query locations: %w", err)
	}
	defer rows.Close()

	var results []models.LocationAverage
	for rows.Next() {
		var r models.LocationAverage
		if err := rows.Scan(
			&r.LocationID, &r.LocationName, &r.LocationAddress,
			&r.Latitude, &r.Longitude, &r.LocationCategory,
			&r.TotalCheckIns, &r.UniqueUsers,
			&r.AvgDownloadMbps, &r.AvgUploadMbps, &r.AvgLatencyMs, &r.AvgJitter,
			&r.SpeedTestCount,
			&r.AvgDecibels, &r.AvgPeakDecibels, &r.NoiseTestCount,
			&r.AvgCrowdedness, &r.AvgEaseOfWork, &r.RatingCount,
			&r.PctOutletsAtBar, &r.PctOutletsAtTable,
			&r.MostCommonWorkType,
			&r.FirstCheckIn, &r.LastCheckIn,
		); err != nil {
			return nil, fmt.Errorf("failed to scan location average: %w", err)
		}
		results = append(results, r)
	}

	if results == nil {
		results = []models.LocationAverage{}
	}

	return results, nil
}
