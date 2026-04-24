package handler

import (
	"fmt"

	"github.com/howsworkingthere/hows-working-there-api/database"

	appErrors "github.com/howsworkingthere/hows-working-there-api/errors"
	"github.com/howsworkingthere/hows-working-there-api/models"
	"gofr.dev/pkg/gofr"
)

// SearchUserByEmail handles GET /api/v1/users/search?email=<email>
// Returns the matching user if found, or 404.
func SearchUserByEmail(c *gofr.Context) (interface{}, error) {
	email := c.Param("email")
	if email == "" {
		return nil, appErrors.BadRequestError{Message: "email query parameter is required"}
	}

	var user models.User
	err := database.DB.QueryRowContext(c,
		`SELECT id, email, user_name, created_at, updated_at FROM users WHERE LOWER(email) = LOWER($1)`,
		email,
	).Scan(&user.ID, &user.Email, &user.UserName, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, appErrors.NotFoundError{Message: fmt.Sprintf("no user found with email: %s", email)}
	}

	return user, nil
}
