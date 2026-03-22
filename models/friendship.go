package models

import "time"

// Friendship represents a row in the friendships table.
type Friendship struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	FriendID  string    `json:"friend_id" db:"friend_id"`
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// FriendProfile is the public-facing friend info returned by the API.
type FriendProfile struct {
	UserID   string `json:"user_id"`
	UserName string `json:"user_name"`
	Email    string `json:"email"`
	Status   string `json:"status"`
}

// FriendRequest is the request body for sending a friend request.
type FriendRequest struct {
	FriendID string `json:"friend_id"`
}

// FriendshipActionRequest is the request body for accepting or denying a friend request.
type FriendshipActionRequest struct {
	FriendshipID string `json:"friendship_id"`
}
