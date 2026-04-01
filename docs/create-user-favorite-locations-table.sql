-- Create user_favorite_locations table for favoriting/bookmarking locations
CREATE TABLE IF NOT EXISTS user_favorite_locations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     TEXT NOT NULL REFERENCES users(id),
    location_id UUID NOT NULL REFERENCES locations(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, location_id)
);

CREATE INDEX IF NOT EXISTS idx_user_favorite_locations_user_id ON user_favorite_locations (user_id);
CREATE INDEX IF NOT EXISTS idx_user_favorite_locations_location_id ON user_favorite_locations (location_id);
