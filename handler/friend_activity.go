package handler

import (
	"fmt"

	appErrors "github.com/howsworkingthere/hows-working-there-api/errors"
	"github.com/howsworkingthere/hows-working-there-api/middleware"
	"github.com/howsworkingthere/hows-working-there-api/models"
	"gofr.dev/pkg/gofr"
)

// GetFriendsActivity handles GET /api/v1/friends/activity
// Returns check-in activity from accepted friends within the last 7 days.
func GetFriendsActivity(c *gofr.Context) (interface{}, error) {
	authedUserID, ok := middleware.GetUserIDFromContext(c.Request.Context())
	if !ok {
		return nil, appErrors.UnauthorizedError{Message: "unable to determine authenticated user"}
	}

	rows, err := c.SQL.QueryContext(c,
		`SELECT u.user_name, l.name, l.address, l.category, ci.timestamp
		 FROM check_ins ci
		 JOIN friendships f ON f.friend_id = ci.user_id AND f.user_id = $1 AND f.status = 'accepted'
		 JOIN users u ON u.id = ci.user_id
		 JOIN locations l ON l.id = ci.location_id
		 WHERE ci.timestamp >= NOW() - INTERVAL '7 days'
		 ORDER BY ci.timestamp DESC
		 LIMIT 30`,
		authedUserID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query friends activity: %w", err)
	}
	defer rows.Close()

	var activities []models.FriendActivity
	for rows.Next() {
		var a models.FriendActivity
		if err := rows.Scan(&a.UserName, &a.LocationName, &a.LocationAddress, &a.Category, &a.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan friend activity: %w", err)
		}
		activities = append(activities, a)
	}

	if activities == nil {
		activities = []models.FriendActivity{}
	}

	return activities, nil
}
