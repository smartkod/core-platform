-- 0002_identity_profile: phone number and account closure.
--
-- Phone is optional: a phone-first market wants it, an e-mail-first one does
-- not. An empty string means "not provided", so the column never forces a fake
-- value and still avoids a nullable unique index.
--
-- closed_at records a user-requested closure before deletion. The row survives
-- so audits and organization membership history stay referentially valid.

ALTER TABLE users ADD COLUMN IF NOT EXISTS phone varchar(32) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone_verified_at timestamptz NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS closed_at timestamptz NULL;

CREATE INDEX IF NOT EXISTS users_phone_idx ON users (phone) WHERE phone <> '';
