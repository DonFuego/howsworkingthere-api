package models

import "time"

// User represents a registered user in the system.
type User struct {
	ID        string    `json:"id" db:"id"`
	Email     string    `json:"email" db:"email"`
	UserName  string    `json:"user_name" db:"user_name"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// RegisterUserRequest represents the request body from Auth0 post-registration trigger.
type RegisterUserRequest struct {
	Data RegisterUserData `json:"data"`
}

// RegisterUserData contains the user fields nested under "data".
type RegisterUserData struct {
	Email    string `json:"email"`
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
}
