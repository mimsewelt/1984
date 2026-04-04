CREATE TABLE posts (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    caption    TEXT         NOT NULL DEFAULT '',
    media_urls TEXT[]       NOT NULL DEFAULT '{}',
    media_type VARCHAR(20)  NOT NULL DEFAULT 'image',
    likes_count    INTEGER  NOT NULL DEFAULT 0,
    comments_count INTEGER  NOT NULL DEFAULT 0,
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_posts_user_id   ON posts (user_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_posts_created   ON posts (created_at DESC)           WHERE deleted_at IS NULL;
