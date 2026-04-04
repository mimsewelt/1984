CREATE TABLE refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT         NOT NULL,
    device_id  VARCHAR(255) NOT NULL DEFAULT 'web',
    expires_at TIMESTAMPTZ  NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user_device ON refresh_tokens (user_id, device_id);
CREATE INDEX idx_refresh_tokens_expires     ON refresh_tokens (expires_at);
