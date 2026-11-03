-- 0005_organization_init: organizations, branches, memberships, regions and
-- service areas.
--
-- Everything a sector-specific engine might want to add (opening hours,
-- capacity, licensing workflow) is deliberately absent. This file holds only
-- what every business has, so an engine can extend it without the core needing
-- to know what the engine does.

CREATE TABLE IF NOT EXISTS organizations (
    id                 uuid PRIMARY KEY,
    slug               varchar(120) NOT NULL UNIQUE,
    name               varchar(160) NOT NULL,
    description        text NOT NULL DEFAULT '',
    logo_url           text NOT NULL DEFAULT '',
    email              varchar(254) NOT NULL DEFAULT '',
    phone              varchar(32)  NOT NULL DEFAULT '',
    website            text NOT NULL DEFAULT '',
    owner_id           uuid NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    status             varchar(30) NOT NULL DEFAULT 'draft',
    verification_level varchar(20) NOT NULL DEFAULT 'none',
    locale             varchar(10) NOT NULL DEFAULT 'en',
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT organizations_status_check
        CHECK (status IN ('draft', 'pending_verification', 'active', 'suspended', 'closed')),
    CONSTRAINT organizations_verification_check
        CHECK (verification_level IN ('none', 'basic', 'verified'))
);

CREATE INDEX IF NOT EXISTS organizations_owner_idx  ON organizations (owner_id);
CREATE INDEX IF NOT EXISTS organizations_status_idx ON organizations (status);
CREATE INDEX IF NOT EXISTS organizations_name_idx   ON organizations (lower(name));

CREATE TABLE IF NOT EXISTS branches (
    id              uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    name            varchar(160) NOT NULL,
    type            varchar(20) NOT NULL DEFAULT 'physical',
    address_line    text NOT NULL DEFAULT '',
    country         varchar(2) NOT NULL DEFAULT '',
    city            varchar(120) NOT NULL DEFAULT '',
    district        varchar(120) NOT NULL DEFAULT '',
    postal_code     varchar(20) NOT NULL DEFAULT '',
    latitude        double precision NULL,
    longitude       double precision NULL,
    phone           varchar(32) NOT NULL DEFAULT '',
    public          boolean NOT NULL DEFAULT true,
    status          varchar(20) NOT NULL DEFAULT 'active',
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT branches_type_check
        CHECK (type IN ('physical', 'service_area', 'online')),
    CONSTRAINT branches_status_check
        CHECK (status IN ('active', 'inactive', 'suspended'))
);

CREATE INDEX IF NOT EXISTS branches_org_idx ON branches (organization_id);
CREATE INDEX IF NOT EXISTS branches_city_idx ON branches (country, city);

CREATE TABLE IF NOT EXISTS memberships (
    id              uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    branch_id       uuid NULL REFERENCES branches (id) ON DELETE SET NULL,
    user_id         uuid NULL REFERENCES users (id) ON DELETE CASCADE,
    invited_email   citext NOT NULL DEFAULT '',
    role_key        varchar(80) NOT NULL,
    status          varchar(20) NOT NULL DEFAULT 'invited',
    invited_by      uuid NULL REFERENCES users (id) ON DELETE SET NULL,
    invited_at      timestamptz NOT NULL DEFAULT now(),
    accepted_at     timestamptz NULL,
    removed_at      timestamptz NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT memberships_status_check
        CHECK (status IN ('invited', 'active', 'suspended', 'removed')),
    -- An active membership must name a user; an invitation must name an address.
    CONSTRAINT memberships_identity_check
        CHECK (user_id IS NOT NULL OR invited_email <> ''),
    -- One person holds one membership per organization.
    CONSTRAINT memberships_org_user_uniq UNIQUE (organization_id, user_id)
);

CREATE INDEX IF NOT EXISTS memberships_org_idx    ON memberships (organization_id);
CREATE INDEX IF NOT EXISTS memberships_user_idx   ON memberships (user_id);
CREATE INDEX IF NOT EXISTS memberships_email_idx  ON memberships (invited_email);

CREATE TABLE IF NOT EXISTS regions (
    id        uuid PRIMARY KEY,
    parent_id uuid NULL REFERENCES regions (id) ON DELETE CASCADE,
    level     varchar(20) NOT NULL,
    code      varchar(32) NOT NULL,
    name      varchar(160) NOT NULL,
    country   varchar(2) NOT NULL DEFAULT '',
    locale    varchar(10) NOT NULL DEFAULT 'en',
    CONSTRAINT regions_level_check
        CHECK (level IN ('country', 'city', 'district')),
    CONSTRAINT regions_code_uniq UNIQUE (country, level, code)
);

CREATE INDEX IF NOT EXISTS regions_parent_idx ON regions (parent_id);

CREATE TABLE IF NOT EXISTS service_areas (
    id              uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    branch_id       uuid NULL REFERENCES branches (id) ON DELETE CASCADE,
    region_id       uuid NULL REFERENCES regions (id) ON DELETE SET NULL,
    mode            varchar(20) NOT NULL DEFAULT 'on_site',
    created_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT service_areas_mode_check
        CHECK (mode IN ('on_site', 'online', 'at_customer', 'hybrid'))
);

CREATE INDEX IF NOT EXISTS service_areas_org_idx ON service_areas (organization_id);

-- Audit of publishing decisions. A status column alone cannot answer who
-- suspended a business or why, and that question is asked during a dispute.
CREATE TABLE IF NOT EXISTS organization_status_events (
    id              uuid PRIMARY KEY,
    organization_id uuid NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    from_status     varchar(30) NOT NULL DEFAULT '',
    to_status       varchar(30) NOT NULL,
    actor_id        uuid NULL REFERENCES users (id) ON DELETE SET NULL,
    reason          text NOT NULL DEFAULT '',
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS organization_status_events_org_idx
    ON organization_status_events (organization_id, created_at DESC);
