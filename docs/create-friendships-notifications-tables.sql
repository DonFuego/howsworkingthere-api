-- Create friendships table for friend connections
-- Two rows per accepted friendship (A→B + B→A); one row while pending (requester→target)
CREATE TABLE IF NOT EXISTS friendships (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    TEXT NOT NULL REFERENCES users(id),
    friend_id  TEXT NOT NULL REFERENCES users(id),
    status     TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'accepted')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, friend_id),
    CHECK (user_id <> friend_id)
);

CREATE INDEX IF NOT EXISTS idx_friendships_user_id ON friendships (user_id);
CREATE INDEX IF NOT EXISTS idx_friendships_friend_id ON friendships (friend_id);
CREATE INDEX IF NOT EXISTS idx_friendships_status ON friendships (user_id, status);

-- Create notifications table for in-app notifications
CREATE TABLE IF NOT EXISTS notifications (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      TEXT NOT NULL REFERENCES users(id),
    type         TEXT NOT NULL CHECK (type IN ('friend_request', 'friend_accepted')),
    from_user_id TEXT NOT NULL REFERENCES users(id),
    reference_id TEXT,
    message      TEXT NOT NULL,
    is_read      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications (user_id, is_read, created_at DESC);
