-- 0001_economy_init: wallet ledger, catalog, orders, payments, entitlements,
-- rewarded ads and reward dials (L2.md rev 1.1 schema).
--
-- Money rule: all amounts are int64 minor units (TOKEN -> 1 jeton = 1,
-- TRY -> kuruş). No float ever stores money.
-- Ledger rule: wallet_entries is append-only. The partial unique index on
-- idempotency_key makes a replay-safe movement impossible to double-write.

CREATE TABLE wallet_accounts (
    id            uuid PRIMARY KEY,
    user_id       uuid NOT NULL,
    currency      varchar(8) NOT NULL,
    balance_minor bigint NOT NULL DEFAULT 0 CHECK (balance_minor >= 0),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, currency)
);

CREATE TABLE wallet_entries (
    id                     uuid PRIMARY KEY,
    account_id             uuid NOT NULL REFERENCES wallet_accounts (id),
    delta_minor            bigint NOT NULL CHECK (delta_minor <> 0),
    balance_after_minor    bigint NOT NULL CHECK (balance_after_minor >= 0),
    reason_code            varchar(40) NOT NULL,
    actor_id               uuid NULL,
    ref_type               varchar(40) NULL,
    ref_id                 uuid NULL,
    idempotency_key        varchar(120) NULL,
    created_at             timestamptz NOT NULL DEFAULT now()
);

-- Double safety net: a stale reader cannot flip the balance negative. Also
-- blocks UPDATE/DELETE via a trigger below, which is what "append-only" means
-- for an audit trail.
CREATE INDEX wallet_entries_account_idx ON wallet_entries (account_id, created_at DESC);

-- Replay-safe movements: the same idempotency key can only produce one entry.
CREATE UNIQUE INDEX wallet_entries_idem_uniq ON wallet_entries (account_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

CREATE OR REPLACE FUNCTION wallet_entries_append_only() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'wallet_entries is append-only (op: %)', TG_OP;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER wallet_entries_no_rewrite
    BEFORE UPDATE OR DELETE ON wallet_entries
    FOR EACH ROW EXECUTE FUNCTION wallet_entries_append_only();

CREATE TABLE products (
    id           uuid PRIMARY KEY,
    code         varchar(60) NOT NULL UNIQUE,
    name         varchar(160) NOT NULL,
    kind         varchar(16) NOT NULL CHECK (kind IN ('token_pack', 'subscription', 'regular')),
    currency     varchar(8) NOT NULL,
    amount_minor bigint NOT NULL CHECK (amount_minor > 0),
    interval     varchar(12) NULL CHECK (interval IN ('day', 'week', 'month', 'year')),
    meta         jsonb NOT NULL DEFAULT '{}'::jsonb,
    active       boolean NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    -- token_pack and subscription need their meta contract; regular needs none.
    CONSTRAINT products_kind_meta CHECK (
        (kind = 'token_pack'  AND jsonb_typeof(meta->'tokens') = 'number') OR
        (kind = 'subscription' AND jsonb_typeof(meta->'entitlement') = 'string') OR
        (kind = 'regular')
    )
);

CREATE TABLE orders (
    id               uuid PRIMARY KEY,
    user_id          uuid NOT NULL,
    product_id       uuid NOT NULL REFERENCES products (id),
    amount_minor     bigint NOT NULL CHECK (amount_minor > 0),
    currency         varchar(8) NOT NULL,
    provider         varchar(40) NOT NULL,
    status           varchar(16) NOT NULL
        CHECK (status IN ('created', 'pending', 'paid', 'fulfilled', 'failed', 'expired', 'refunded')),
    idempotency_key  varchar(120) NOT NULL UNIQUE,
    reference_code   varchar(40) NOT NULL UNIQUE,
    expires_at       timestamptz NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX orders_user_idx ON orders (user_id, created_at DESC);

CREATE TABLE payments (
    id            uuid PRIMARY KEY,
    order_id      uuid NOT NULL REFERENCES orders (id),
    provider      varchar(40) NOT NULL,
    provider_ref  varchar(120) NULL,
    status        varchar(16) NOT NULL CHECK (status IN ('pending', 'paid', 'failed')),
    amount_minor  bigint NOT NULL CHECK (amount_minor > 0),
    currency      varchar(8) NOT NULL,
    paid_at       timestamptz NULL,
    failure_code  varchar(40) NULL,
    raw_payload   jsonb NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_ref)
);

CREATE TABLE payment_events (
    id           uuid PRIMARY KEY,
    provider     varchar(40) NOT NULL,
    event_id     varchar(120) NOT NULL,
    order_id     uuid NOT NULL REFERENCES orders (id),
    type         varchar(40) NOT NULL,
    payload      jsonb NULL,
    processed_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, event_id)
);

CREATE TABLE entitlements (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL,
    product_id uuid NULL REFERENCES products (id),
    code       varchar(80) NOT NULL,
    source     varchar(40) NOT NULL,
    starts_at  timestamptz NOT NULL,
    ends_at    timestamptz NULL,
    revoked_at timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX entitlements_user_active_idx ON entitlements (user_id, code)
    WHERE revoked_at IS NULL;
-- Entitlement idempotency is per order_id (checked in code): a renewed
-- subscription legitimately grants a second active row.

CREATE TABLE refunds (
    id           uuid PRIMARY KEY,
    payment_id   uuid NOT NULL REFERENCES payments (id),
    amount_minor bigint NOT NULL CHECK (amount_minor > 0),
    provider_ref varchar(120) NULL,
    status       varchar(16) NOT NULL CHECK (status IN ('pending', 'done', 'failed')),
    reason       varchar(160) NOT NULL DEFAULT '',
    created_by   uuid NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------
-- Rewarded ads (L2.md rev 1.1 §3.11)
-- ---------------------------------------------------------------

CREATE TABLE ads (
    id            uuid PRIMARY KEY,
    code          varchar(60) NOT NULL UNIQUE,
    title         varchar(160) NOT NULL,
    media_type    varchar(8) NOT NULL CHECK (media_type IN ('video', 'image', 'text')),
    body_text     text NULL,
    media_url     text NULL,
    content       text NULL,
    target_url    text NULL,
    watch_seconds int NOT NULL CHECK (watch_seconds > 0),
    watch_reward  bigint NOT NULL CHECK (watch_reward >= 0),
    click_reward  bigint NOT NULL CHECK (click_reward >= 0),
    starts_at     timestamptz NULL,
    ends_at       timestamptz NULL,
    active        boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ads_media_shape CHECK (
        (media_type = 'text'  AND body_text IS NOT NULL AND media_url IS NULL) OR
        (media_type <> 'text' AND media_url IS NOT NULL)
    ),
    CONSTRAINT ads_target_scheme CHECK (
        target_url IS NULL OR target_url LIKE 'https://%'
    )
);

CREATE INDEX ads_active_rewarded_idx ON ads (active)
    WHERE active AND watch_reward > 0;

CREATE TABLE ads_clicks (
    id          uuid PRIMARY KEY,
    ad_id       uuid NOT NULL REFERENCES ads (id) ON DELETE CASCADE,
    user_id     uuid NULL,
    kind        varchar(8) NOT NULL CHECK (kind IN ('click', 'view_redirect')),
    reward_date date NULL,
    ip_hash     varchar(64) NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- One rewarded click per user per ad per day; anonymous clicks (user_id NULL)
-- cost nothing and count only as statistics.
CREATE UNIQUE INDEX ads_clicks_reward_uniq ON ads_clicks (user_id, ad_id, reward_date)
    WHERE kind = 'click' AND user_id IS NOT NULL AND reward_date IS NOT NULL;

CREATE INDEX ads_clicks_ad_idx ON ads_clicks (ad_id, created_at DESC);

CREATE TABLE ad_view_sessions (
    id                uuid PRIMARY KEY,
    ad_id             uuid NOT NULL REFERENCES ads (id) ON DELETE CASCADE,
    user_id           uuid NOT NULL,
    required_seconds  int NOT NULL CHECK (required_seconds > 0),
    status            varchar(10) NOT NULL CHECK (status IN ('started', 'rewarded')),
    server_started_at timestamptz NOT NULL,
    expires_at        timestamptz NOT NULL,
    reward_minor      bigint NOT NULL CHECK (reward_minor >= 0),
    reward_date       date NOT NULL,
    entry_id          uuid NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    -- One earning session per (user, ad, day): farming one ad all day is
    -- structurally impossible, not just application policy.
    UNIQUE (user_id, ad_id, reward_date)
);

CREATE INDEX ad_view_sessions_user_idx ON ad_view_sessions (user_id, reward_date);

-- ---------------------------------------------------------------
-- Economy dials (L2.md rev 1.1 §3.13): one row, admin-editable.
-- Config seeds row 1; the panel changes it after that.
-- ---------------------------------------------------------------

CREATE TABLE reward_settings (
    id                   int PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    signup_grant         bigint NOT NULL DEFAULT 10 CHECK (signup_grant >= 0),
    ad_watch_reward      bigint NOT NULL DEFAULT 1 CHECK (ad_watch_reward >= 0),
    daily_reward_cap     bigint NOT NULL DEFAULT 3 CHECK (daily_reward_cap >= 0),
    monthly_reward_cap   bigint NOT NULL DEFAULT 10 CHECK (monthly_reward_cap >= 0),
    click_reward_enabled boolean NOT NULL DEFAULT true,
    updated_at           timestamptz NOT NULL DEFAULT now(),
    updated_by           uuid NULL
);
