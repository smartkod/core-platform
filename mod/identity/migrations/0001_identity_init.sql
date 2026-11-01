-- 0001_identity_init: identity schema.
--
-- All timestamps are timestamptz and written in UTC. IDs are UUIDv7 generated
-- by the application so an ID is known before INSERT (needed for idempotency
-- and event publishing).

CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
    id                uuid PRIMARY KEY,
    email             citext NOT NULL UNIQUE,
    password_hash     text   NOT NULL,
    name              varchar(120) NOT NULL DEFAULT '',
    locale            varchar(10)  NOT NULL DEFAULT 'en',
    avatar_url        text   NOT NULL DEFAULT '',
    role              varchar(20)  NOT NULL DEFAULT 'user',
    status            varchar(20)  NOT NULL DEFAULT 'active',
    email_verified_at timestamptz NULL,
    last_login_at     timestamptz NULL,
    token_version     integer NOT NULL DEFAULT 0,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    deleted_at        timestamptz NULL
);

CREATE INDEX users_created_at_idx ON users (created_at DESC);
CREATE INDEX users_deleted_at_idx ON users (deleted_at) WHERE deleted_at IS NOT NULL;

-- Refresh tokens are stored hashed: a database leak must not open sessions.
-- family_id groups every token descending from one login so a reuse detection
-- can revoke the whole family at once.
CREATE TABLE refresh_tokens (
    id          uuid PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash  char(64) NOT NULL UNIQUE,
    family_id   uuid NOT NULL,
    device      varchar(120) NOT NULL DEFAULT '',
    ip          varchar(64)  NOT NULL DEFAULT '',
    audience    varchar(20)  NOT NULL DEFAULT 'mobile',
    expires_at  timestamptz NOT NULL,
    revoked_at  timestamptz NULL,
    replaced_by uuid NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX refresh_tokens_user_idx   ON refresh_tokens (user_id);
CREATE INDEX refresh_tokens_family_idx ON refresh_tokens (family_id);
CREATE INDEX refresh_tokens_expiry_idx ON refresh_tokens (expires_at);

-- Single-use, hashed tokens for e-mail verification and password reset.
CREATE TABLE one_time_tokens (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    purpose    varchar(20) NOT NULL,          -- verify_email | reset_password
    token_hash char(64) NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    used_at    timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX one_time_tokens_user_idx ON one_time_tokens (user_id, purpose);
CREATE INDEX one_time_tokens_expiry_idx ON one_time_tokens (expires_at);
