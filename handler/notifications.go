package handler

import (
	"fmt"

	"github.com/howsworkingthere/hows-working-there-api/database"

	appErrors "github.com/howsworkingthere/hows-working-there-api/errors"
	"github.com/howsworkingthere/hows-working-there-api/middleware"
	"github.com/howsworkingthere/hows-working-there-api/models"
	"gofr.dev/pkg/gofr"
)

// ListNotifications handles GET /api/v1/notifications
// Returns all notifications for the current user, newest first.
func ListNotifications(c *gofr.Context) (interface{}, error) {
	authedUserID, ok := middleware.GetUserIDFromContext(c.Request.Context())
	if !ok {
		return nil, appErrors.UnauthorizedError{Message: "unable to determine authenticated user"}
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(c,
		`SELECT n.id, n.type, n.from_user_id, u.user_name, n.reference_id, n.message, n.is_read, n.created_at
		 FROM notifications n
		 JOIN users u ON u.id = n.from_user_id
		 WHERE n.user_id = $1
		 ORDER BY n.created_at DESC
		 LIMIT 50`,
		authedUserID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query notifications: %w", err)
	}
	defer rows.Close()

	var notifications []models.NotificationWithSender
	for rows.Next() {
		var n models.NotificationWithSender
		if err := rows.Scan(&n.ID, &n.Type, &n.FromUserID, &n.FromUserName, &n.ReferenceID, &n.Message, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, n)
	}

	if notifications == nil {
		notifications = []models.NotificationWithSender{}
	}

	return notifications, nil
}

// MarkNotificationRead handles POST /api/v1/notifications/{id}/read
// Marks a single notification as read.
func MarkNotificationRead(c *gofr.Context) (interface{}, error) {
	authedUserID, ok := middleware.GetUserIDFromContext(c.Request.Context())
	if !ok {
		return nil, appErrors.UnauthorizedError{Message: "unable to determine authenticated user"}
	}

	notificationID := c.PathParam("id")
	if notificationID == "" {
		return nil, appErrors.BadRequestError{Message: "notification id path parameter is required"}
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(c,
		`UPDATE notifications SET is_read = TRUE WHERE id = $1 AND user_id = $2`,
		notificationID, authedUserID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to mark notification as read: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return nil, appErrors.NotFoundError{Message: "notification not found"}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit mark read: %w", err)
	}

	return map[string]string{"status": "read"}, nil
}
