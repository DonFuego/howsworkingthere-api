package models

// UserLocationAverage maps to the v_user_location_averages view.
type UserLocationAverage struct {
	UserID                 string   `json:"user_id" db:"user_id"`
	LocationID             string   `json:"location_id" db:"location_id"`
	LocationName           string   `json:"location_name" db:"location_name"`
	LocationAddress        string   `json:"location_address" db:"location_address"`
	Latitude               float64  `json:"latitude" db:"latitude"`
	Longitude              float64  `json:"longitude" db:"longitude"`
	LocationCategory       string   `json:"location_category" db:"location_category"`
	MyCheckIns             int      `json:"my_check_ins" db:"my_check_ins"`
	MyAvgDownloadMbps      *float64 `json:"my_avg_download_mbps" db:"my_avg_download_mbps"`
	MyAvgUploadMbps        *float64 `json:"my_avg_upload_mbps" db:"my_avg_upload_mbps"`
	MyAvgLatencyMs         *float64 `json:"my_avg_latency_ms" db:"my_avg_latency_ms"`
	MyAvgJitter            *float64 `json:"my_avg_jitter" db:"my_avg_jitter"`
	MySpeedTestCount       int      `json:"my_speed_test_count" db:"my_speed_test_count"`
	MyAvgDecibels          *float64 `json:"my_avg_decibels" db:"my_avg_decibels"`
	MyAvgPeakDecibels      *float64 `json:"my_avg_peak_decibels" db:"my_avg_peak_decibels"`
	MyNoiseTestCount       int      `json:"my_noise_test_count" db:"my_noise_test_count"`
	MyAvgCrowdedness       *float64 `json:"my_avg_crowdedness" db:"my_avg_crowdedness"`
	MyAvgEaseOfWork        *float64 `json:"my_avg_ease_of_work" db:"my_avg_ease_of_work"`
	MyRatingCount          int      `json:"my_rating_count" db:"my_rating_count"`
	MyPctOutletsAtBar      *float64 `json:"my_pct_outlets_at_bar" db:"my_pct_outlets_at_bar"`
	MyPctOutletsAtTable    *float64 `json:"my_pct_outlets_at_table" db:"my_pct_outlets_at_table"`
	MyMostCommonWorkType   *string  `json:"my_most_common_work_type" db:"my_most_common_work_type"`
	MyMostCommonEaseOfWork *int     `json:"my_most_common_ease_of_work" db:"my_most_common_ease_of_work"`
	MyFirstCheckIn         *string  `json:"my_first_check_in" db:"my_first_check_in"`
	MyLastCheckIn          *string  `json:"my_last_check_in" db:"my_last_check_in"`
}

// LocationAverage maps to the v_location_averages view.
type LocationAverage struct {
	LocationID           string   `json:"location_id" db:"location_id"`
	LocationName         string   `json:"location_name" db:"location_name"`
	LocationAddress      string   `json:"location_address" db:"location_address"`
	Latitude             float64  `json:"latitude" db:"latitude"`
	Longitude            float64  `json:"longitude" db:"longitude"`
	LocationCategory     string   `json:"location_category" db:"location_category"`
	TotalCheckIns        int      `json:"total_check_ins" db:"total_check_ins"`
	UniqueUsers          int      `json:"unique_users" db:"unique_users"`
	AvgDownloadMbps      *float64 `json:"avg_download_mbps" db:"avg_download_mbps"`
	AvgUploadMbps        *float64 `json:"avg_upload_mbps" db:"avg_upload_mbps"`
	AvgLatencyMs         *float64 `json:"avg_latency_ms" db:"avg_latency_ms"`
	AvgJitter            *float64 `json:"avg_jitter" db:"avg_jitter"`
	SpeedTestCount       int      `json:"speed_test_count" db:"speed_test_count"`
	AvgDecibels          *float64 `json:"avg_decibels" db:"avg_decibels"`
	AvgPeakDecibels      *float64 `json:"avg_peak_decibels" db:"avg_peak_decibels"`
	NoiseTestCount       int      `json:"noise_test_count" db:"noise_test_count"`
	AvgCrowdedness       *float64 `json:"avg_crowdedness" db:"avg_crowdedness"`
	AvgEaseOfWork        *float64 `json:"avg_ease_of_work" db:"avg_ease_of_work"`
	RatingCount          int      `json:"rating_count" db:"rating_count"`
	PctOutletsAtBar      *float64 `json:"pct_outlets_at_bar" db:"pct_outlets_at_bar"`
	PctOutletsAtTable    *float64 `json:"pct_outlets_at_table" db:"pct_outlets_at_table"`
	MostCommonWorkType   *string  `json:"most_common_work_type" db:"most_common_work_type"`
	MostCommonEaseOfWork *int     `json:"most_common_ease_of_work" db:"most_common_ease_of_work"`
	FirstCheckIn         *string  `json:"first_check_in" db:"first_check_in"`
	LastCheckIn          *string  `json:"last_check_in" db:"last_check_in"`
}
