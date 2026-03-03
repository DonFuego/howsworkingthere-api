package models

import "time"

// Location represents a physical place in the database.
type Location struct {
	ID                string    `json:"id" db:"id"`
	Name              string    `json:"name" db:"name"`
	Address           string    `json:"address" db:"address"`
	Latitude          float64   `json:"latitude" db:"latitude"`
	Longitude         float64   `json:"longitude" db:"longitude"`
	Category          string    `json:"category" db:"category"`
	MapkitPOICategory *string   `json:"mapkit_poi_category,omitempty" db:"mapkit_poi_category"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// LocationRequest is the location portion of a check-in payload.
type LocationRequest struct {
	Name              string  `json:"name" validate:"required"`
	Address           string  `json:"address" validate:"required"`
	Latitude          float64 `json:"latitude" validate:"required"`
	Longitude         float64 `json:"longitude" validate:"required"`
	Category          string  `json:"category" validate:"required"`
	MapkitPOICategory *string `json:"mapkit_poi_category"`
}
