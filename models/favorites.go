package models

import "time"

// UserFavoriteLocation represents a user's favorited location.
type UserFavoriteLocation struct {
	ID               string    `json:"id" db:"id"`
	UserID           string    `json:"user_id" db:"user_id"`
	LocationID       string    `json:"location_id" db:"location_id"`
	LocationName     string    `json:"location_name" db:"location_name"`
	LocationAddress  string    `json:"location_address" db:"location_address"`
	LocationCategory string    `json:"location_category" db:"location_category"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// AddFavoriteRequest is the JSON body for POST /api/v1/favorites.
type AddFavoriteRequest struct {
	LocationID string `json:"location_id"`
}

// IsFavoritedResponse is the JSON response for GET /api/v1/favorites/{location_id}.
type IsFavoritedResponse struct {
	IsFavorited bool `json:"is_favorited"`
}
