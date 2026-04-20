package handler

import (
	"fmt"
	"strconv"
	"strings"

	appErrors "github.com/howsworkingthere/hows-working-there-api/errors"
	"github.com/howsworkingthere/hows-working-there-api/models"
	"gofr.dev/pkg/gofr"
)

// GetTrendingLocations handles GET /api/v1/locations/trending
// Returns the top N locations ranked by number of check-ins within the requested time window.
// Query params:
//   - period: "today" | "week" | "month" (default: "week")
//   - limit:  1-50 (default: 5)
func GetTrendingLocations(c *gofr.Context) (interface{}, error) {
	period := strings.ToLower(strings.TrimSpace(c.Param("period")))
	if period == "" {
		period = "week"
	}

	var interval string
	switch period {
	case "today":
		// Midnight of today in the database's timezone
		interval = "CURRENT_DATE"
	case "week":
		interval = "NOW() - INTERVAL '7 days'"
	case "month":
		interval = "NOW() - INTERVAL '30 days'"
	default:
		return nil, appErrors.BadRequestError{Message: "invalid period: must be 'today', 'week', or 'month'"}
	}

	limit := 5
	if limitStr := c.Param("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	query := fmt.Sprintf(`
		SELECT
			l.id,
			l.name,
			l.address,
			l.latitude,
			l.longitude,
			l.category,
			COUNT(ci.id) AS check_ins,
			COUNT(DISTINCT ci.user_id) AS unique_users,
			ROUND(AVG(ci.work_score), 0) AS avg_work_score
		FROM locations l
		INNER JOIN check_ins ci ON ci.location_id = l.id
		WHERE ci.timestamp >= %s
		GROUP BY l.id, l.name, l.address, l.latitude, l.longitude, l.category
		ORDER BY check_ins DESC, unique_users DESC, l.name ASC
		LIMIT %d`, interval, limit)

	rows, err := c.SQL.QueryContext(c, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query trending locations: %w", err)
	}
	defer rows.Close()

	results := []models.TrendingLocation{}
	rank := 0
	for rows.Next() {
		var r models.TrendingLocation
		if err := rows.Scan(
			&r.LocationID, &r.LocationName, &r.LocationAddress,
			&r.Latitude, &r.Longitude, &r.LocationCategory,
			&r.CheckIns, &r.UniqueUsers, &r.AvgWorkScore,
		); err != nil {
			return nil, fmt.Errorf("failed to scan trending location: %w", err)
		}
		rank++
		r.Rank = rank
		results = append(results, r)
	}

	return map[string]interface{}{
		"period":    period,
		"locations": results,
	}, nil
}
