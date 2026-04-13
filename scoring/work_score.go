package scoring

import (
	"time"

	"github.com/howsworkingthere/hows-working-there-api/models"
)

// ComputeWorkScore calculates a 0-100 work quality score from check-in data.
// Returns the total score as an integer.
func ComputeWorkScore(speed *models.SpeedTestRequest, noise *models.NoiseLevelRequest, workspace *models.WorkspaceRatingsRequest) int {
	score := 0
	score += networkSpeedScore(speed)
	score += noiseScore(noise)
	score += easeOfWorkScore(workspace)
	score += outletScore(workspace)
	score += crowdednessScore(workspace)
	score += workTypeScore(workspace)
	return score
}

// TimeOfDay returns the time-of-day bucket for a given timestamp.
// Morning: 6:00-11:59, Afternoon: 12:00-17:59, Evening: 18:00-5:59
func TimeOfDay(t time.Time) string {
	hour := t.UTC().Hour()
	switch {
	case hour >= 6 && hour < 12:
		return "morning"
	case hour >= 12 && hour < 18:
		return "afternoon"
	default:
		return "evening"
	}
}

// networkSpeedScore returns 0-30 points based on download, upload, and jitter.
func networkSpeedScore(st *models.SpeedTestRequest) int {
	if st == nil || st.Skipped {
		return 0
	}

	// Download sub-score (0-12)
	dl := st.DownloadSpeedMbps
	var dlScore int
	switch {
	case dl >= 100:
		dlScore = 12
	case dl >= 50:
		dlScore = 10
	case dl >= 25:
		dlScore = 8
	case dl >= 10:
		dlScore = 5
	case dl >= 5:
		dlScore = 3
	default:
		dlScore = 1
	}

	// Upload sub-score (0-10)
	ul := st.UploadSpeedMbps
	var ulScore int
	switch {
	case ul >= 50:
		ulScore = 10
	case ul >= 20:
		ulScore = 8
	case ul >= 10:
		ulScore = 6
	case ul >= 5:
		ulScore = 4
	default:
		ulScore = 1
	}

	// Jitter sub-score (0-8)
	j := st.Jitter
	var jScore int
	switch {
	case j < 2:
		jScore = 8
	case j < 5:
		jScore = 6
	case j < 15:
		jScore = 4
	case j < 30:
		jScore = 2
	default:
		jScore = 1
	}

	return dlScore + ulScore + jScore
}

// noiseScore returns 0-20 points based on average decibel level.
func noiseScore(nl *models.NoiseLevelRequest) int {
	if nl == nil || nl.Skipped {
		return 0
	}

	db := nl.AverageDecibels
	switch {
	case db <= 30:
		return 20
	case db <= 40:
		return 17
	case db <= 50:
		return 14
	case db <= 60:
		return 10
	case db <= 70:
		return 6
	case db <= 80:
		return 3
	default:
		return 1
	}
}

// easeOfWorkScore returns 0-18 points based on ease of work rating.
func easeOfWorkScore(wr *models.WorkspaceRatingsRequest) int {
	if wr == nil {
		return 0
	}
	switch wr.EaseOfWork {
	case 1:
		return 18 // Easy
	case 2:
		return 10 // Moderate
	case 3:
		return 3 // Difficult
	default:
		return 0
	}
}

// outletScore returns 0-14 points based on outlet availability.
func outletScore(wr *models.WorkspaceRatingsRequest) int {
	if wr == nil {
		return 0
	}
	switch {
	case wr.OutletsAtBar && wr.OutletsAtTable:
		return 14
	case wr.OutletsAtBar || wr.OutletsAtTable:
		return 8
	default:
		return 2
	}
}

// crowdednessScore returns 0-12 points based on crowdedness rating.
func crowdednessScore(wr *models.WorkspaceRatingsRequest) int {
	if wr == nil {
		return 0
	}
	switch wr.Crowdedness {
	case 1:
		return 12 // Empty
	case 2:
		return 7 // Somewhat Crowded
	case 3:
		return 2 // Crowded
	default:
		return 0
	}
}

// workTypeScore returns 0-6 points based on best work type.
func workTypeScore(wr *models.WorkspaceRatingsRequest) int {
	if wr == nil {
		return 0
	}
	switch wr.BestWorkType {
	case "team":
		return 6
	case "solo":
		return 3
	default:
		return 0
	}
}
