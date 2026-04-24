package handler

import (
	"database/sql"
	"fmt"

	"github.com/howsworkingthere/hows-working-there-api/database"

	appErrors "github.com/howsworkingthere/hows-working-there-api/errors"
	"github.com/howsworkingthere/hows-working-there-api/models"
	"gofr.dev/pkg/gofr"
)

// GetLocationScore handles GET /api/v1/locations/{location_id}/score
// Returns the aggregated work score summary for a location with time-of-day breakdowns.
func GetLocationScore(c *gofr.Context) (interface{}, error) {
	locationID := c.PathParam("location_id")
	if locationID == "" {
		return nil, appErrors.BadRequestError{Message: "location_id path parameter is required"}
	}

	// Verify the location exists
	var locExists bool
	err := database.DB.QueryRowContext(c,
		`SELECT EXISTS(SELECT 1 FROM locations WHERE id = $1)`, locationID,
	).Scan(&locExists)
	if err != nil || !locExists {
		return nil, appErrors.NotFoundError{Message: fmt.Sprintf("location not found: %s", locationID)}
	}

	query := fmt.Sprintf(`
	SELECT
		ROUND(AVG(work_score), 0) AS avg_work_score,
		COUNT(*) FILTER (WHERE work_score IS NOT NULL) AS scored_check_ins,
		ROUND(AVG(work_score) FILTER (WHERE time_of_day = 'morning'), 0) AS avg_score_morning,
		COUNT(*) FILTER (WHERE time_of_day = 'morning' AND work_score IS NOT NULL) AS morning_check_ins,
		ROUND(AVG(work_score) FILTER (WHERE time_of_day = 'afternoon'), 0) AS avg_score_afternoon,
		COUNT(*) FILTER (WHERE time_of_day = 'afternoon' AND work_score IS NOT NULL) AS afternoon_check_ins,
		ROUND(AVG(work_score) FILTER (WHERE time_of_day = 'evening'), 0) AS avg_score_evening,
		COUNT(*) FILTER (WHERE time_of_day = 'evening' AND work_score IS NOT NULL) AS evening_check_ins
	FROM check_ins
	WHERE location_id = '%s'`, locationID)

	var summary models.WorkScoreSummary
	var avgScoreMorning, avgScoreAfternoon, avgScoreEvening *float64
	var morningCheckIns, afternoonCheckIns, eveningCheckIns int

	err = database.DB.QueryRowContext(c, query).Scan(
		&summary.OverallScore, &summary.TotalCheckIns,
		&avgScoreMorning, &morningCheckIns,
		&avgScoreAfternoon, &afternoonCheckIns,
		&avgScoreEvening, &eveningCheckIns,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.WorkScoreSummary{}, nil
		}
		return nil, fmt.Errorf("failed to query location score: %w", err)
	}

	if avgScoreMorning != nil && morningCheckIns > 0 {
		summary.ByTimeOfDay.Morning = &models.TimeOfDayBucket{Score: *avgScoreMorning, CheckIns: morningCheckIns}
	}
	if avgScoreAfternoon != nil && afternoonCheckIns > 0 {
		summary.ByTimeOfDay.Afternoon = &models.TimeOfDayBucket{Score: *avgScoreAfternoon, CheckIns: afternoonCheckIns}
	}
	if avgScoreEvening != nil && eveningCheckIns > 0 {
		summary.ByTimeOfDay.Evening = &models.TimeOfDayBucket{Score: *avgScoreEvening, CheckIns: eveningCheckIns}
	}

	return map[string]interface{}{
		"location_id": locationID,
		"work_score":  summary,
	}, nil
}
