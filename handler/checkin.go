package handler

import (
	"database/sql"
	"fmt"

	"github.com/howsworkingthere/hows-working-there-api/database"
	appErrors "github.com/howsworkingthere/hows-working-there-api/errors"
	"github.com/howsworkingthere/hows-working-there-api/middleware"
	"github.com/howsworkingthere/hows-working-there-api/models"
	"github.com/howsworkingthere/hows-working-there-api/scoring"
	"gofr.dev/pkg/gofr"
)

// CreateCheckIn handles POST /api/v1/check-ins
// Creates a check-in linked to an existing location, with speed test, noise level, and workspace ratings.
func CreateCheckIn(c *gofr.Context) (interface{}, error) {
	var req models.CheckInRequest
	if err := c.Bind(&req); err != nil {
		return nil, fmt.Errorf("invalid request body: %w", err)
	}

	if req.ID == "" || req.UserID == "" || req.Location.ID == "" {
		return nil, fmt.Errorf("missing required fields: id, user_id, and location.id are required")
	}

	// Verify the request user_id matches the authenticated user
	authedUserID, ok := middleware.GetUserIDFromContext(c.Request.Context())
	if !ok {
		return nil, fmt.Errorf("unable to determine authenticated user")
	}
	if req.UserID != authedUserID {
		return nil, appErrors.ForbiddenError{Message: "user_id mismatch: token sub does not match request user_id"}
	}

	locationID := req.Location.ID

	// Verify the location exists
	var locExists bool
	err := database.DB.QueryRowContext(c,
		`SELECT EXISTS(SELECT 1 FROM locations WHERE id = $1)`, locationID,
	).Scan(&locExists)
	if err != nil || !locExists {
		return nil, appErrors.NotFoundError{Message: fmt.Sprintf("location not found: %s", locationID)}
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check idempotency: if check-in with this ID already exists, return it
	var existingCheckInID string
	err = tx.QueryRowContext(c,
		`SELECT id FROM check_ins WHERE id = $1`, req.ID,
	).Scan(&existingCheckInID)
	if err == nil {
		// Already exists — return existing record
		tx.Rollback()
		return models.CheckInResponse{
			CheckInID:  existingCheckInID,
			LocationID: locationID,
			CreatedAt:  req.Timestamp,
		}, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check idempotency: %w", err)
	}

	// Insert check-in
	_, err = tx.ExecContext(c,
		`INSERT INTO check_ins (id, user_id, location_id, timestamp) VALUES ($1, $2, $3, $4)`,
		req.ID, req.UserID, locationID, req.Timestamp,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert check-in: %w", err)
	}

	// Insert speed test
	if err := insertSpeedTest(c, tx, req.ID, locationID, &req.SpeedTest); err != nil {
		return nil, err
	}

	// Insert noise level
	if err := insertNoiseLevel(c, tx, req.ID, locationID, &req.NoiseLevel); err != nil {
		return nil, err
	}

	// Insert workspace ratings
	if err := insertWorkspaceRatings(c, tx, req.ID, locationID, &req.WorkspaceRatings); err != nil {
		return nil, err
	}

	// Compute and store work score
	workScore := scoring.ComputeWorkScore(&req.SpeedTest, &req.NoiseLevel, &req.WorkspaceRatings)
	timeOfDay := scoring.TimeOfDay(req.Timestamp)
	_, err = tx.ExecContext(c,
		`UPDATE check_ins SET work_score = $1, time_of_day = $2 WHERE id = $3`,
		workScore, timeOfDay, req.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update work score: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return models.CheckInResponse{
		CheckInID:  req.ID,
		LocationID: locationID,
		CreatedAt:  req.Timestamp,
	}, nil
}

// CreateCheckInAtLocation handles POST /api/v1/locations/{location_id}/check-ins
// Creates a check-in at an existing location with speed test, noise level, and workspace ratings.
func CreateCheckInAtLocation(c *gofr.Context) (interface{}, error) {
	locationID := c.PathParam("location_id")
	if locationID == "" {
		return nil, fmt.Errorf("location_id path parameter is required")
	}

	var req models.ExistingLocationCheckInRequest
	if err := c.Bind(&req); err != nil {
		return nil, fmt.Errorf("invalid request body: %w", err)
	}

	if req.ID == "" || req.UserID == "" {
		return nil, fmt.Errorf("missing required fields: id and user_id are required")
	}

	// Verify the request user_id matches the authenticated user
	authedUserID, ok := middleware.GetUserIDFromContext(c.Request.Context())
	if !ok {
		return nil, fmt.Errorf("unable to determine authenticated user")
	}
	if req.UserID != authedUserID {
		return nil, appErrors.ForbiddenError{Message: "user_id mismatch: token sub does not match request user_id"}
	}

	// Verify the location exists
	var locExists bool
	err := database.DB.QueryRowContext(c,
		`SELECT EXISTS(SELECT 1 FROM locations WHERE id = $1)`, locationID,
	).Scan(&locExists)
	if err != nil || !locExists {
		return nil, appErrors.NotFoundError{Message: fmt.Sprintf("location not found: %s", locationID)}
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Check idempotency
	var existingCheckInID string
	err = tx.QueryRowContext(c,
		`SELECT id FROM check_ins WHERE id = $1`, req.ID,
	).Scan(&existingCheckInID)
	if err == nil {
		tx.Rollback()
		return models.CheckInResponse{
			CheckInID:  existingCheckInID,
			LocationID: locationID,
			CreatedAt:  req.Timestamp,
		}, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check idempotency: %w", err)
	}

	// Insert check-in
	_, err = tx.ExecContext(c,
		`INSERT INTO check_ins (id, user_id, location_id, timestamp) VALUES ($1, $2, $3, $4)`,
		req.ID, req.UserID, locationID, req.Timestamp,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert check-in: %w", err)
	}

	// Insert speed test
	if err := insertSpeedTest(c, tx, req.ID, locationID, &req.SpeedTest); err != nil {
		return nil, err
	}

	// Insert noise level
	if err := insertNoiseLevel(c, tx, req.ID, locationID, &req.NoiseLevel); err != nil {
		return nil, err
	}

	// Insert workspace ratings
	if err := insertWorkspaceRatings(c, tx, req.ID, locationID, &req.WorkspaceRatings); err != nil {
		return nil, err
	}

	// Compute and store work score
	workScore := scoring.ComputeWorkScore(&req.SpeedTest, &req.NoiseLevel, &req.WorkspaceRatings)
	timeOfDay := scoring.TimeOfDay(req.Timestamp)
	_, err = tx.ExecContext(c,
		`UPDATE check_ins SET work_score = $1, time_of_day = $2 WHERE id = $3`,
		workScore, timeOfDay, req.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update work score: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return models.CheckInResponse{
		CheckInID:  req.ID,
		LocationID: locationID,
		CreatedAt:  req.Timestamp,
	}, nil
}

func insertSpeedTest(c *gofr.Context, tx *sql.Tx, checkInID, locationID string, st *models.SpeedTestRequest) error {
	var serverDomain, serverCity, serverCountry *string
	if st.Server != nil {
		serverDomain = st.Server.Domain
		serverCity = st.Server.City
		serverCountry = st.Server.Country
	}

	_, err := tx.ExecContext(c,
		`INSERT INTO speed_tests (
			check_in_id, location_id,
			download_speed_mbps, upload_speed_mbps, latency_ms, jitter,
			isp_name, ip_address, network_type, packet_loss_percent,
			time_to_first_byte_ms, download_transferred_mb, upload_transferred_mb,
			server_domain, server_city, server_country,
			speed_test_id, skipped
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`,
		checkInID, locationID,
		st.DownloadSpeedMbps, st.UploadSpeedMbps, st.LatencyMs, st.Jitter,
		st.ISPName, st.IPAddress, st.NetworkType, st.PacketLossPercent,
		st.TimeToFirstByteMs, st.DownloadTransferredMB, st.UploadTransferredMB,
		serverDomain, serverCity, serverCountry,
		st.SpeedTestID, st.Skipped,
	)
	if err != nil {
		return fmt.Errorf("failed to insert speed test: %w", err)
	}
	return nil
}

func insertNoiseLevel(c *gofr.Context, tx *sql.Tx, checkInID, locationID string, nl *models.NoiseLevelRequest) error {
	_, err := tx.ExecContext(c,
		`INSERT INTO noise_levels (
			check_in_id, location_id,
			average_decibels, peak_decibels, duration_seconds, skipped
		) VALUES ($1, $2, $3, $4, $5, $6)`,
		checkInID, locationID,
		nl.AverageDecibels, nl.PeakDecibels, nl.DurationSeconds, nl.Skipped,
	)
	if err != nil {
		return fmt.Errorf("failed to insert noise level: %w", err)
	}
	return nil
}

func insertWorkspaceRatings(c *gofr.Context, tx *sql.Tx, checkInID, locationID string, wr *models.WorkspaceRatingsRequest) error {
	_, err := tx.ExecContext(c,
		`INSERT INTO workspace_ratings (
			check_in_id, location_id,
			outlets_at_bar, outlets_at_table,
			crowdedness, ease_of_work, best_work_type
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		checkInID, locationID,
		wr.OutletsAtBar, wr.OutletsAtTable,
		wr.Crowdedness, wr.EaseOfWork, wr.BestWorkType,
	)
	if err != nil {
		return fmt.Errorf("failed to insert workspace ratings: %w", err)
	}
	return nil
}
