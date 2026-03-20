package models

import "time"

// LocationDetailResponse is the top-level response for GET /api/v1/locations/{location_id}/detail.
type LocationDetailResponse struct {
	Location        Location              `json:"location"`
	TotalCheckIns   int                   `json:"total_check_ins"`
	UniqueUsers     int                   `json:"unique_users"`
	FirstCheckIn    *time.Time            `json:"first_check_in"`
	LastCheckIn     *time.Time            `json:"last_check_in"`
	Noise           NoiseStats            `json:"noise"`
	SpeedByISP      []ISPSpeedStats       `json:"speed_by_isp"`
	WorkspaceRating WorkspaceDistribution `json:"workspace_ratings"`
}

// NoiseStats holds averaged noise measurements for a location.
type NoiseStats struct {
	AvgDecibels     *float64 `json:"avg_decibels"`
	AvgPeakDecibels *float64 `json:"avg_peak_decibels"`
	TestCount       int      `json:"test_count"`
}

// ISPSpeedStats holds averaged speed test results for a single ISP at a location.
type ISPSpeedStats struct {
	ISPName        string   `json:"isp_name"`
	AvgDownloadMbps *float64 `json:"avg_download_mbps"`
	AvgUploadMbps   *float64 `json:"avg_upload_mbps"`
	AvgLatencyMs    *float64 `json:"avg_latency_ms"`
	AvgJitter       *float64 `json:"avg_jitter"`
	TestCount      int      `json:"test_count"`
}

// RatingDistribution holds percentages for a 3-value rating dimension.
type RatingDistribution struct {
	Level1 float64 `json:"level_1"`
	Level2 float64 `json:"level_2"`
	Level3 float64 `json:"level_3"`
}

// WorkTypeDistribution holds percentages for work type categories.
type WorkTypeDistribution struct {
	Solo float64 `json:"solo"`
	Team float64 `json:"team"`
	Both float64 `json:"both"`
}

// CrowdednessDistribution holds percentages for crowdedness levels.
type CrowdednessDistribution struct {
	Empty           float64 `json:"empty"`
	SomewhatCrowded float64 `json:"somewhat_crowded"`
	Crowded         float64 `json:"crowded"`
}

// EaseOfWorkDistribution holds percentages for ease of work levels.
type EaseOfWorkDistribution struct {
	Easy      float64 `json:"easy"`
	Moderate  float64 `json:"moderate"`
	Difficult float64 `json:"difficult"`
}

// WorkspaceDistribution holds all workspace rating distributions for a location.
type WorkspaceDistribution struct {
	TotalRatings    int                     `json:"total_ratings"`
	PctOutletsAtBar  float64                `json:"pct_outlets_at_bar"`
	PctOutletsAtTable float64               `json:"pct_outlets_at_table"`
	Crowdedness      CrowdednessDistribution `json:"crowdedness"`
	EaseOfWork       EaseOfWorkDistribution  `json:"ease_of_work"`
	BestWorkType     WorkTypeDistribution    `json:"best_work_type"`
}
