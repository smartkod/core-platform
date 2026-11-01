-- 0002_files_init: file storage metadata.
--
-- The bytes live in object storage; this table is the index. The owner column is
-- not nullable and every user-facing query filters on it, so an authorization
-- mistake in one handler cannot silently expose another user's files.

CREATE TABLE files (
    id            uuid PRIMARY KEY,
    owner_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    object_key    text NOT NULL UNIQUE,
    original_name varchar(255) NOT NULL DEFAULT '',
    content_type  varchar(120) NOT NULL,
    size_bytes    bigint NOT NULL CHECK (size_bytes >= 0),
    checksum      char(64) NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz NULL
);

-- Listing one owner's files newest-first is the only hot query.
CREATE INDEX files_owner_created_idx ON files (owner_id, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX files_deleted_at_idx ON files (deleted_at) WHERE deleted_at IS NOT NULL;
