-- 0003_notifications_init: notification outbox, preferences and read state.
--
-- The outbox is the reason this table exists. A message is written here first
-- and delivered by a worker afterwards, so a mail server that is down delays a
-- notification instead of losing it. The rendered subject and body are stored,
-- not just the template name: a retry must send what was originally intended
-- even if the template changes in between.

CREATE TABLE notifications (
    id               uuid PRIMARY KEY,
    user_id          uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    channel          varchar(20) NOT NULL,
    template         varchar(80) NOT NULL DEFAULT '',
    subject          varchar(255) NOT NULL DEFAULT '',
    body             text NOT NULL DEFAULT '',
    payload          jsonb NOT NULL DEFAULT '{}',
    status           varchar(20) NOT NULL DEFAULT 'pending',
    attempts         integer NOT NULL DEFAULT 0,
    last_error       text NOT NULL DEFAULT '',
    next_attempt_at  timestamptz NOT NULL DEFAULT now(),
    read_at          timestamptz NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    sent_at          timestamptz NULL
);

-- The worker's only query: pending rows that are due.
CREATE INDEX notifications_due_idx ON notifications (next_attempt_at)
    WHERE status = 'pending';

-- The user's inbox: their in-app messages, newest first.
CREATE INDEX notifications_inbox_idx ON notifications (user_id, created_at DESC)
    WHERE channel = 'in_app';

-- Unread badge.
CREATE INDEX notifications_unread_idx ON notifications (user_id)
    WHERE channel = 'in_app' AND read_at IS NULL;

CREATE TABLE notification_preferences (
    user_id    uuid PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    channels   jsonb NOT NULL DEFAULT '{}',
    updated_at timestamptz NOT NULL DEFAULT now()
);
