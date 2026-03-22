package models

import "time"

// Notification represents a row in the notifications table.
type Notification struct {
	ID         string    `json:"id" db:"id"`
	UserID     string    `json:"user_id" db:"user_id"`
	Type       string    `json:"type" db:"type"`
	FromUserID string    `json:"from_user_id" db:"from_user_id"`
	ReferenceID *string  `json:"reference_id,omitempty" db:"reference_id"`
	Message    string    `json:"message" db:"message"`
	IsRead     bool      `json:"is_read" db:"is_read"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

// NotificationWithSender includes the sender's name for display.
type NotificationWithSender struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	FromUserID   string    `json:"from_user_id"`
	FromUserName string    `json:"from_user_name"`
	ReferenceID  *string   `json:"reference_id,omitempty"`
	Message      string    `json:"message"`
	IsRead       bool      `json:"is_read"`
	CreatedAt    time.Time `json:"created_at"`
}
