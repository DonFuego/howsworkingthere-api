package handler

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/howsworkingthere/hows-working-there-api/database"

	apierrors "github.com/howsworkingthere/hows-working-there-api/errors"
	"github.com/howsworkingthere/hows-working-there-api/models"
	"gofr.dev/pkg/gofr"
)

// CreateLocation handles POST /api/v1/locations
// Inserts a new location and returns the full row including the generated UUID.
func CreateLocation(c *gofr.Context) (interface{}, error) {
	var req models.LocationRequest
	if err := c.Bind(&req); err != nil {
		return nil, apierrors.BadRequestError{Message: "invalid request body: " + err.Error()}
	}

	if req.Name == "" || req.Address == "" || req.Category == "" {
		return nil, apierrors.BadRequestError{Message: "name, address, and category are required"}
	}

	var loc models.Location
	err := database.DB.QueryRowContext(c,
		`INSERT INTO locations (name, address, latitude, longitude, category, mapkit_poi_category)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, name, address, latitude, longitude, category, mapkit_poi_category, created_at, updated_at`,
		req.Name, req.Address, req.Latitude, req.Longitude, req.Category, req.MapkitPOICategory,
	).Scan(
		&loc.ID, &loc.Name, &loc.Address,
		&loc.Latitude, &loc.Longitude,
		&loc.Category, &loc.MapkitPOICategory,
		&loc.CreatedAt, &loc.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "uq_locations_name_coords") {
			return nil, apierrors.BadRequestError{Message: "a location with this name and coordinates already exists"}
		}
		return nil, fmt.Errorf("failed to insert location: %w", err)
	}

	return loc, nil
}

// SearchLocations handles GET /api/v1/locations/search
// Supports searching by name, address, or geo-coordinates (latitude + longitude + optional radius_km).
func SearchLocations(c *gofr.Context) (interface{}, error) {
	name := strings.TrimSpace(c.Param("name"))
	address := strings.TrimSpace(c.Param("address"))
	latStr := c.Param("latitude")
	lngStr := c.Param("longitude")
	radiusStr := c.Param("radius_km")

	hasName := name != ""
	hasAddress := address != ""
	hasGeo := latStr != "" && lngStr != ""

	if !hasName && !hasAddress && !hasGeo {
		return nil, fmt.Errorf("at least one search parameter required: name, address, or latitude+longitude")
	}

	var conditions []string
	var args []interface{}
	argIdx := 1

	if hasName {
		conditions = append(conditions, fmt.Sprintf("LOWER(name) LIKE LOWER('%%' || $%d || '%%')", argIdx))
		args = append(args, name)
		argIdx++
	}

	if hasAddress {
		conditions = append(conditions, fmt.Sprintf("LOWER(address) LIKE LOWER('%%' || $%d || '%%')", argIdx))
		args = append(args, address)
		argIdx++
	}

	if hasGeo {
		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid latitude value")
		}
		lng, err := strconv.ParseFloat(lngStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid longitude value")
		}

		radiusKm := 0.5
		if radiusStr != "" {
			r, err := strconv.ParseFloat(radiusStr, 64)
			if err == nil && r > 0 {
				radiusKm = r
			}
		}

		haversine := fmt.Sprintf(`(6371 * acos(LEAST(1.0, cos(radians($%d)) * cos(radians(latitude)) * cos(radians(longitude) - radians($%d)) + sin(radians($%d)) * sin(radians(latitude))))) <= $%d`, argIdx, argIdx+1, argIdx+2, argIdx+3)
		conditions = append(conditions, haversine)
		args = append(args, lat, lng, lat, radiusKm)
		argIdx += 4
	}

	query := "SELECT id, name, address, latitude, longitude, category, mapkit_poi_category, created_at, updated_at FROM locations"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY name LIMIT 50"

	rows, err := database.DB.QueryContext(c, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search locations: %w", err)
	}
	defer rows.Close()

	var locations []models.Location
	for rows.Next() {
		var loc models.Location
		if err := rows.Scan(
			&loc.ID, &loc.Name, &loc.Address,
			&loc.Latitude, &loc.Longitude,
			&loc.Category, &loc.MapkitPOICategory,
			&loc.CreatedAt, &loc.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan location: %w", err)
		}
		locations = append(locations, loc)
	}

	if locations == nil {
		locations = []models.Location{}
	}

	return locations, nil
}
