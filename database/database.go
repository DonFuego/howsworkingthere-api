// Package database provides a pgx-backed *sql.DB that is safe to use through
// PgBouncer / Supavisor in transaction-pooling mode.
//
// The key detail is that the underlying pgx connection is configured with
// QueryExecModeExec, which bypasses the implicit prepared-statement cache that
// lib/pq uses. That cache was causing intermittent
// "pq: unnamed prepared statement does not exist (26000)" failures when running
// behind the Supabase/DO connection pooler.
package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// DB is the shared database handle used by all API handlers.
// It is initialized by Init() during application startup.
var DB *sql.DB

// Init opens the database connection using pgx's stdlib driver with
// QueryExecModeExec so that queries work through PgBouncer transaction pools.
//
// Configuration is read from the same environment variables GoFr uses for
// its own Supabase datasource so we stay compatible with the existing
// deployment (.do/app.yaml):
//   - DB_USER, DB_PASSWORD, DB_NAME, DB_SSL_MODE
//   - SUPABASE_PROJECT_REF, SUPABASE_REGION, SUPABASE_CONNECTION_TYPE
//
// For a non-Supabase deploy, DB_HOST and DB_PORT are honored directly.
func Init() error {
	dsn, err := buildDSN()
	if err != nil {
		return err
	}

	pgxConfig, err := pgx.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("database: failed to parse pgx config: %w", err)
	}

	// Bypass the implicit prepared-statement cache. This is what makes the
	// connection safe for PgBouncer / Supavisor transaction pooling.
	pgxConfig.DefaultQueryExecMode = pgx.QueryExecModeExec

	connStr := stdlib.RegisterConnConfig(pgxConfig)
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return fmt.Errorf("database: failed to open pgx pool: %w", err)
	}

	// Reasonable defaults for a mobile-backed API.
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return fmt.Errorf("database: ping failed: %w", err)
	}

	DB = db
	log.Printf("[database] connected via pgx (QueryExecModeExec) to %s:%s", pgxConfig.Host, portString(pgxConfig.Port))

	return nil
}

// Close closes the underlying pool. Safe to call even if Init was never called.
func Close() error {
	if DB == nil {
		return nil
	}
	return DB.Close()
}

// buildDSN constructs a libpq-style DSN from environment variables, honoring
// the Supabase-specific variables when DB_DIALECT=supabase.
func buildDSN() (string, error) {
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbName := strings.TrimSpace(os.Getenv("DB_NAME"))
	if dbName == "" {
		dbName = "postgres"
	}
	sslMode := strings.TrimSpace(os.Getenv("DB_SSL_MODE"))
	if sslMode == "" {
		sslMode = "require"
	}

	host := strings.TrimSpace(os.Getenv("DB_HOST"))
	port := strings.TrimSpace(os.Getenv("DB_PORT"))

	dialect := strings.ToLower(strings.TrimSpace(os.Getenv("DB_DIALECT")))
	if dialect == "supabase" {
		projectRef := strings.TrimSpace(os.Getenv("SUPABASE_PROJECT_REF"))
		region := strings.TrimSpace(os.Getenv("SUPABASE_REGION"))
		connType := strings.ToLower(strings.TrimSpace(os.Getenv("SUPABASE_CONNECTION_TYPE")))
		if connType == "" {
			connType = "session"
		}

		switch connType {
		case "direct":
			host = fmt.Sprintf("db.%s.supabase.co", projectRef)
			port = "5432"
			// user stays as-is (typically "postgres")
		case "session":
			host = fmt.Sprintf("aws-0-%s.pooler.supabase.co", region)
			port = "5432"
			user = fmt.Sprintf("postgres.%s", projectRef)
		case "transaction":
			host = fmt.Sprintf("aws-0-%s.pooler.supabase.co", region)
			port = "6543"
			user = fmt.Sprintf("postgres.%s", projectRef)
		default:
			return "", fmt.Errorf("database: unknown SUPABASE_CONNECTION_TYPE %q", connType)
		}
	}

	if host == "" {
		return "", fmt.Errorf("database: DB_HOST is not set")
	}
	if port == "" {
		port = "5432"
	}
	if user == "" {
		return "", fmt.Errorf("database: DB_USER is not set")
	}

	// libpq / pgx accept this URL form.
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		user, password, host, port, dbName, sslMode,
	)

	return dsn, nil
}

func portString(port uint16) string {
	return fmt.Sprintf("%d", port)
}
