package handler

import (
	"fmt"

	appErrors "github.com/howsworkingthere/hows-working-there-api/errors"
	"github.com/howsworkingthere/hows-working-there-api/middleware"
	"github.com/howsworkingthere/hows-working-there-api/models"
	"gofr.dev/pkg/gofr"
)

// SendFriendRequest handles POST /api/v1/friends/request
// Creates a pending friendship row and a notification for the target user.
func SendFriendRequest(c *gofr.Context) (interface{}, error) {
	authedUserID, ok := middleware.GetUserIDFromContext(c.Request.Context())
	if !ok {
		return nil, appErrors.UnauthorizedError{Message: "unable to determine authenticated user"}
	}

	var req models.FriendRequest
	if err := c.Bind(&req); err != nil {
		return nil, appErrors.BadRequestError{Message: fmt.Sprintf("invalid request body: %v", err)}
	}

	if req.FriendID == "" {
		return nil, appErrors.BadRequestError{Message: "friend_id is required"}
	}

	if req.FriendID == authedUserID {
		return nil, appErrors.BadRequestError{Message: "you cannot send a friend request to yourself"}
	}

	// Verify target user exists
	var targetExists bool
	err := c.SQL.QueryRowContext(c,
		`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, req.FriendID,
	).Scan(&targetExists)
	if err != nil || !targetExists {
		return nil, appErrors.NotFoundError{Message: "target user not found"}
	}

	// Check if friendship already exists in either direction
	var existingCount int
	err = c.SQL.QueryRowContext(c,
		`SELECT COUNT(*) FROM friendships
		 WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)`,
		authedUserID, req.FriendID,
	).Scan(&existingCount)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing friendship: %w", err)
	}
	if existingCount > 0 {
		return nil, appErrors.BadRequestError{Message: "friendship or pending request already exists"}
	}

	// Get requester's name for notification message
	var requesterName string
	err = c.SQL.QueryRowContext(c,
		`SELECT user_name FROM users WHERE id = $1`, authedUserID,
	).Scan(&requesterName)
	if err != nil {
		requesterName = "Someone"
	}

	tx, err := c.SQL.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Create pending friendship (requester → target)
	var friendshipID string
	err = tx.QueryRowContext(c,
		`INSERT INTO friendships (user_id, friend_id, status)
		 VALUES ($1, $2, 'pending')
		 RETURNING id`,
		authedUserID, req.FriendID,
	).Scan(&friendshipID)
	if err != nil {
		return nil, fmt.Errorf("failed to create friend request: %w", err)
	}

	// Create notification for the target user
	_, err = tx.ExecContext(c,
		`INSERT INTO notifications (user_id, type, from_user_id, reference_id, message)
		 VALUES ($1, 'friend_request', $2, $3, $4)`,
		req.FriendID, authedUserID, friendshipID,
		fmt.Sprintf("%s sent you a friend request", requesterName),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit friend request: %w", err)
	}

	return map[string]string{
		"status":        "pending",
		"friendship_id": friendshipID,
	}, nil
}

// AcceptFriendRequest handles POST /api/v1/friends/accept
// Accepts a pending friend request: flips status to accepted and creates the reciprocal row.
func AcceptFriendRequest(c *gofr.Context) (interface{}, error) {
	authedUserID, ok := middleware.GetUserIDFromContext(c.Request.Context())
	if !ok {
		return nil, appErrors.UnauthorizedError{Message: "unable to determine authenticated user"}
	}

	var req models.FriendshipActionRequest
	if err := c.Bind(&req); err != nil {
		return nil, appErrors.BadRequestError{Message: fmt.Sprintf("invalid request body: %v", err)}
	}

	if req.FriendshipID == "" {
		return nil, appErrors.BadRequestError{Message: "friendship_id is required"}
	}

	// Verify this is a pending request TO the current user
	var friendship models.Friendship
	err := c.SQL.QueryRowContext(c,
		`SELECT id, user_id, friend_id, status FROM friendships WHERE id = $1`,
		req.FriendshipID,
	).Scan(&friendship.ID, &friendship.UserID, &friendship.FriendID, &friendship.Status)
	if err != nil {
		return nil, appErrors.NotFoundError{Message: "friendship not found"}
	}

	if friendship.FriendID != authedUserID {
		return nil, appErrors.ForbiddenError{Message: "you can only accept requests sent to you"}
	}
	if friendship.Status != "pending" {
		return nil, appErrors.BadRequestError{Message: "this request is not pending"}
	}

	// Get acceptor's name for notification
	var acceptorName string
	err = c.SQL.QueryRowContext(c,
		`SELECT user_name FROM users WHERE id = $1`, authedUserID,
	).Scan(&acceptorName)
	if err != nil {
		acceptorName = "Someone"
	}

	tx, err := c.SQL.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update existing row to accepted
	_, err = tx.ExecContext(c,
		`UPDATE friendships SET status = 'accepted', updated_at = NOW() WHERE id = $1`,
		req.FriendshipID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to accept friend request: %w", err)
	}

	// Create reciprocal row (accepted)
	_, err = tx.ExecContext(c,
		`INSERT INTO friendships (user_id, friend_id, status)
		 VALUES ($1, $2, 'accepted')
		 ON CONFLICT (user_id, friend_id) DO UPDATE SET status = 'accepted', updated_at = NOW()`,
		authedUserID, friendship.UserID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create reciprocal friendship: %w", err)
	}

	// Notify the original requester
	_, err = tx.ExecContext(c,
		`INSERT INTO notifications (user_id, type, from_user_id, reference_id, message)
		 VALUES ($1, 'friend_accepted', $2, $3, $4)`,
		friendship.UserID, authedUserID, req.FriendshipID,
		fmt.Sprintf("%s accepted your friend request", acceptorName),
	)
	if err != nil {
		// Non-critical — log but don't fail
		c.Logger.Errorf("failed to create acceptance notification: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit accept: %w", err)
	}

	return map[string]string{"status": "accepted"}, nil
}

// DenyFriendRequest handles POST /api/v1/friends/deny
// Denies a pending friend request: deletes the pending row.
func DenyFriendRequest(c *gofr.Context) (interface{}, error) {
	authedUserID, ok := middleware.GetUserIDFromContext(c.Request.Context())
	if !ok {
		return nil, appErrors.UnauthorizedError{Message: "unable to determine authenticated user"}
	}

	var req models.FriendshipActionRequest
	if err := c.Bind(&req); err != nil {
		return nil, appErrors.BadRequestError{Message: fmt.Sprintf("invalid request body: %v", err)}
	}

	if req.FriendshipID == "" {
		return nil, appErrors.BadRequestError{Message: "friendship_id is required"}
	}

	// Verify this is a pending request TO the current user
	var friendID, status string
	err := c.SQL.QueryRowContext(c,
		`SELECT friend_id, status FROM friendships WHERE id = $1`,
		req.FriendshipID,
	).Scan(&friendID, &status)
	if err != nil {
		return nil, appErrors.NotFoundError{Message: "friendship not found"}
	}

	if friendID != authedUserID {
		return nil, appErrors.ForbiddenError{Message: "you can only deny requests sent to you"}
	}
	if status != "pending" {
		return nil, appErrors.BadRequestError{Message: "this request is not pending"}
	}

	tx, err := c.SQL.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete the pending row
	_, err = tx.ExecContext(c,
		`DELETE FROM friendships WHERE id = $1`, req.FriendshipID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to deny friend request: %w", err)
	}

	// Mark related notifications as read
	_, _ = tx.ExecContext(c,
		`UPDATE notifications SET is_read = TRUE WHERE reference_id = $1 AND user_id = $2`,
		req.FriendshipID, authedUserID,
	)

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit deny: %w", err)
	}

	return map[string]string{"status": "denied"}, nil
}

// RemoveFriend handles DELETE /api/v1/friends/{friend_id}
// Removes both friendship rows between the two users.
func RemoveFriend(c *gofr.Context) (interface{}, error) {
	authedUserID, ok := middleware.GetUserIDFromContext(c.Request.Context())
	if !ok {
		return nil, appErrors.UnauthorizedError{Message: "unable to determine authenticated user"}
	}

	friendID := c.PathParam("friend_id")
	if friendID == "" {
		return nil, appErrors.BadRequestError{Message: "friend_id path parameter is required"}
	}

	tx, err := c.SQL.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(c,
		`DELETE FROM friendships
		 WHERE (user_id = $1 AND friend_id = $2) OR (user_id = $2 AND friend_id = $1)`,
		authedUserID, friendID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to remove friend: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, appErrors.NotFoundError{Message: "friendship not found"}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit remove: %w", err)
	}

	return map[string]string{"status": "removed"}, nil
}

// ListFriends handles GET /api/v1/friends
// Returns all friends (accepted) and pending outgoing/incoming requests for the current user.
func ListFriends(c *gofr.Context) (interface{}, error) {
	authedUserID, ok := middleware.GetUserIDFromContext(c.Request.Context())
	if !ok {
		return nil, appErrors.UnauthorizedError{Message: "unable to determine authenticated user"}
	}

	rows, err := c.SQL.QueryContext(c,
		`SELECT u.id, u.user_name, u.email, f.status
		 FROM friendships f
		 JOIN users u ON u.id = f.friend_id
		 WHERE f.user_id = $1
		 ORDER BY f.status ASC, u.user_name ASC`,
		authedUserID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query friends: %w", err)
	}
	defer rows.Close()

	var friends []models.FriendProfile
	for rows.Next() {
		var f models.FriendProfile
		if err := rows.Scan(&f.UserID, &f.UserName, &f.Email, &f.Status); err != nil {
			return nil, fmt.Errorf("failed to scan friend: %w", err)
		}
		friends = append(friends, f)
	}

	if friends == nil {
		friends = []models.FriendProfile{}
	}

	return friends, nil
}
