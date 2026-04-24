package handler

import (
	"fmt"

	"github.com/howsworkingthere/hows-working-there-api/database"

	appErrors "github.com/howsworkingthere/hows-working-there-api/errors"
	"github.com/howsworkingthere/hows-working-there-api/middleware"
	"github.com/howsworkingthere/hows-working-there-api/models"
	"gofr.dev/pkg/gofr"
)

// AddFavorite handles POST /api/v1/favorites
// Adds a location to the authenticated user's favorites.
func AddFavorite(c *gofr.Context) (interface{}, error) {
	authedUserID, ok := middleware.GetUserIDFromContext(c.Request.Context())
	if !ok {
		return nil, appErrors.UnauthorizedError{Message: "unable to determine authenticated user"}
	}

	var req models.AddFavoriteRequest
	if err := c.Bind(&req); err != nil {
		return nil, appErrors.BadRequestError{Message: fmt.Sprintf("invalid request body: %v", err)}
	}

	if req.LocationID == "" {
		return nil, appErrors.BadRequestError{Message: "location_id is required"}
	}

	// Upsert — ignore conflict if already favorited
	var fav models.UserFavoriteLocation
	err := database.DB.QueryRowContext(c,
		`INSERT INTO user_favorite_locations (user_id, location_id)
		 VALUES ($1, $2)
		 ON CONFLICT (user_id, location_id) DO UPDATE SET user_id = EXCLUDED.user_id
		 RETURNING id, user_id, location_id, created_at`,
		authedUserID, req.LocationID,
	).Scan(&fav.ID, &fav.UserID, &fav.LocationID, &fav.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to add favorite: %w", err)
	}

	return fav, nil
}

// RemoveFavorite handles DELETE /api/v1/favorites/{location_id}
// Removes a location from the authenticated user's favorites.
func RemoveFavorite(c *gofr.Context) (interface{}, error) {
	authedUserID, ok := middleware.GetUserIDFromContext(c.Request.Context())
	if !ok {
		return nil, appErrors.UnauthorizedError{Message: "unable to determine authenticated user"}
	}

	locationID := c.PathParam("location_id")
	if locationID == "" {
		return nil, appErrors.BadRequestError{Message: "location_id path parameter is required"}
	}

	result, err := database.DB.ExecContext(c,
		`DELETE FROM user_favorite_locations WHERE user_id = $1 AND location_id = $2`,
		authedUserID, locationID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to remove favorite: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return nil, appErrors.NotFoundError{Message: "favorite not found"}
	}

	return map[string]string{"status": "removed"}, nil
}

// ListFavorites handles GET /api/v1/favorites
// Returns all favorited locations for the authenticated user.
func ListFavorites(c *gofr.Context) (interface{}, error) {
	authedUserID, ok := middleware.GetUserIDFromContext(c.Request.Context())
	if !ok {
		return nil, appErrors.UnauthorizedError{Message: "unable to determine authenticated user"}
	}

	rows, err := database.DB.QueryContext(c,
		`SELECT f.id, f.user_id, f.location_id,
		        l.name AS location_name, l.address AS location_address, l.category AS location_category,
		        f.created_at
		 FROM user_favorite_locations f
		 JOIN locations l ON l.id = f.location_id
		 WHERE f.user_id = $1
		 ORDER BY f.created_at DESC`,
		authedUserID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list favorites: %w", err)
	}
	defer rows.Close()

	var results []models.UserFavoriteLocation
	for rows.Next() {
		var f models.UserFavoriteLocation
		if err := rows.Scan(
			&f.ID, &f.UserID, &f.LocationID,
			&f.LocationName, &f.LocationAddress, &f.LocationCategory,
			&f.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan favorite: %w", err)
		}
		results = append(results, f)
	}

	if results == nil {
		results = []models.UserFavoriteLocation{}
	}

	return results, nil
}

// CheckFavorite handles GET /api/v1/favorites/{location_id}
// Returns whether the authenticated user has favorited a specific location.
func CheckFavorite(c *gofr.Context) (interface{}, error) {
	authedUserID, ok := middleware.GetUserIDFromContext(c.Request.Context())
	if !ok {
		return nil, appErrors.UnauthorizedError{Message: "unable to determine authenticated user"}
	}

	locationID := c.PathParam("location_id")
	if locationID == "" {
		return nil, appErrors.BadRequestError{Message: "location_id path parameter is required"}
	}

	var exists bool
	err := database.DB.QueryRowContext(c,
		`SELECT EXISTS(SELECT 1 FROM user_favorite_locations WHERE user_id = $1 AND location_id = $2)`,
		authedUserID, locationID,
	).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check favorite: %w", err)
	}

	return models.IsFavoritedResponse{IsFavorited: exists}, nil
}
