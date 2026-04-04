CREATE TABLE signal_key_bundles (
    user_id          UUID    PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    identity_key_dh  BYTEA   NOT NULL,
    identity_key_sig BYTEA   NOT NULL,
    signed_pre_key   BYTEA   NOT NULL,
    spk_signature    BYTEA   NOT NULL,
    spk_key_id       INTEGER NOT NULL DEFAULT 1,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE one_time_prekeys (
    id         UUID    PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID    NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_id     INTEGER NOT NULL,
    public_key BYTEA   NOT NULL,
    used       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_opk_user_unused ON one_time_prekeys (user_id) WHERE used = FALSE;

CREATE TABLE conversations (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE conversation_members (
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id)         ON DELETE CASCADE,
    joined_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (conversation_id, user_id)
);

CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID    NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       UUID    NOT NULL REFERENCES users(id)         ON DELETE CASCADE,
    ciphertext      BYTEA   NOT NULL,
    message_type    VARCHAR(20) NOT NULL DEFAULT 'signal',
    sent_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivered_at    TIMESTAMPTZ,
    read_at         TIMESTAMPTZ
);

CREATE INDEX idx_messages_conversation ON messages (conversation_id, sent_at DESC);
CREATE INDEX idx_messages_sender       ON messages (sender_id, sent_at DESC);
