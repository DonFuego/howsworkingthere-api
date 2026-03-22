package models

import "time"

// FriendActivity represents a friend's recent check-in for the activity feed.
type FriendActivity struct {
	UserName        string    `json:"user_name"`
	LocationID      string    `json:"location_id"`
	LocationName    string    `json:"location_name"`
	LocationAddress string    `json:"location_address"`
	Category        string    `json:"category"`
	Timestamp       time.Time `json:"timestamp"`
}
