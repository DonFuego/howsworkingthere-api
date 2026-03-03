package models

import "time"

// CheckInRequest is the full payload sent by the iOS client for a new check-in.
type CheckInRequest struct {
	ID               string                  `json:"id" validate:"required"`
	UserID           string                  `json:"user_id" validate:"required"`
	Timestamp        time.Time               `json:"timestamp" validate:"required"`
	Location         LocationRequest         `json:"location" validate:"required"`
	SpeedTest        SpeedTestRequest        `json:"speed_test" validate:"required"`
	NoiseLevel       NoiseLevelRequest       `json:"noise_level" validate:"required"`
	WorkspaceRatings WorkspaceRatingsRequest `json:"workspace_ratings" validate:"required"`
}

// ExistingLocationCheckInRequest is the payload for a check-in at an already-known location.
type ExistingLocationCheckInRequest struct {
	ID               string                  `json:"id" validate:"required"`
	UserID           string                  `json:"user_id" validate:"required"`
	Timestamp        time.Time               `json:"timestamp" validate:"required"`
	SpeedTest        SpeedTestRequest        `json:"speed_test" validate:"required"`
	NoiseLevel       NoiseLevelRequest       `json:"noise_level" validate:"required"`
	WorkspaceRatings WorkspaceRatingsRequest `json:"workspace_ratings" validate:"required"`
}

// CheckInResponse is returned after successfully creating a check-in.
type CheckInResponse struct {
	CheckInID     string    `json:"check_in_id"`
	LocationID    string    `json:"location_id"`
	LocationIsNew bool      `json:"location_is_new,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}
