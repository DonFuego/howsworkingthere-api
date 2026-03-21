package handler

import (
	"fmt"

	appErrors "github.com/howsworkingthere/hows-working-there-api/errors"
	"github.com/howsworkingthere/hows-working-there-api/models"
	"gofr.dev/pkg/gofr"
)

// RegisterUser handles POST /user/register
// Called by Auth0 post-registration trigger to create a user record.
// JWT validation (HMAC with AUTH0_TRIGGER_SECRET) is handled by Auth0Middleware.
func RegisterUser(c *gofr.Context) (interface{}, error) {
	var req models.RegisterUserRequest
	if err := c.Bind(&req); err != nil {
		return nil, appErrors.BadRequestError{Message: fmt.Sprintf("invalid request body: %v", err)}
	}

	data := req.Data
	if data.Email == "" || data.UserID == "" || data.UserName == "" {
		return nil, appErrors.BadRequestError{Message: "missing required fields: email, user_id, and user_name are required"}
	}

	_, err := c.SQL.ExecContext(c,
		`INSERT INTO users (id, email, user_name)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (id) DO NOTHING`,
		data.UserID, data.Email, data.UserName,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	return map[string]string{
		"status":  "created",
		"user_id": data.UserID,
	}, nil
}
