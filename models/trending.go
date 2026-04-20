package models

// TrendingLocation represents a location ranked by check-in activity within a time window.
type TrendingLocation struct {
	LocationID       string   `json:"id" db:"id"`
	LocationName     string   `json:"name" db:"name"`
	LocationAddress  string   `json:"address" db:"address"`
	Latitude         float64  `json:"latitude" db:"latitude"`
	Longitude        float64  `json:"longitude" db:"longitude"`
	LocationCategory string   `json:"category" db:"category"`
	CheckIns         int      `json:"check_ins" db:"check_ins"`
	UniqueUsers      int      `json:"unique_users" db:"unique_users"`
	AvgWorkScore     *float64 `json:"avg_work_score" db:"avg_work_score"`
	Rank             int      `json:"rank" db:"rank"`
}
