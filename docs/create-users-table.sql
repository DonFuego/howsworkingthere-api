-- Create users table for Auth0 post-registration webhook
CREATE TABLE IF NOT EXISTS users (
    id         TEXT PRIMARY KEY,          -- Auth0 user_id (e.g. "auth0|abc123")
    email      TEXT NOT NULL,
    user_name  TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);
