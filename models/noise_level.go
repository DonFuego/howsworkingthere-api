package models

// NoiseLevelRequest is the noise level portion of a check-in payload.
type NoiseLevelRequest struct {
	AverageDecibels float64 `json:"average_decibels"`
	PeakDecibels    float64 `json:"peak_decibels"`
	DurationSeconds float64 `json:"duration_seconds"`
	Skipped         bool    `json:"skipped"`
}
