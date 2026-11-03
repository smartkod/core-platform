-- 0006_catalog_init: taxonomy, organization categories, offerings and variants.
--
-- The catalogue answers "what does this business offer" and nothing else.
-- Duration, capacity, stock, SKU and pricing rules are engine concerns: storing
-- them here would force every future engine into one shape.

CREATE TABLE IF NOT EXISTS taxonomy_nodes (
    id                   uuid PRIMARY KEY,
    parent_id            uuid NULL REFERENCES taxonomy_nodes (id) ON DELETE RESTRICT,
    slug                 varchar(120) NOT NULL UNIQUE,
    name                 varchar(160) NOT NULL,
    description          text NOT NULL DEFAULT '',
    status               varchar(20) NOT NULL DEFAULT 'active',
    published            boolean NOT NULL DEFAULT true,
    sort                 integer NOT NULL DEFAULT 0,
    locale               varchar(10) NOT NULL DEFAULT 'en',
    extension_schema_ref varchar(160) NOT NULL DEFAULT '',
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT taxonomy_nodes_status_check
        CHECK (status IN ('active', 'inactive'))
);

CREATE INDEX IF NOT EXISTS taxonomy_nodes_parent_idx ON taxonomy_nodes (parent_id);

-- An organization may belong to several categories. The pair is the key, so a
-- duplicate never appears and removal is exact.
CREATE TABLE IF NOT EXISTS organization_categories (
    organization_id uuid NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    taxonomy_id     uuid NOT NULL REFERENCES taxonomy_nodes (id) ON DELETE CASCADE,
    created_at      timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (organization_id, taxonomy_id)
);

CREATE INDEX IF NOT EXISTS organization_categories_taxonomy_idx
    ON organization_categories (taxonomy_id);

CREATE TABLE IF NOT EXISTS offerings (
    id               uuid PRIMARY KEY,
    organization_id  uuid NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    branch_id        uuid NULL REFERENCES branches (id) ON DELETE SET NULL,
    taxonomy_id      uuid NOT NULL REFERENCES taxonomy_nodes (id) ON DELETE RESTRICT,
    name             varchar(200) NOT NULL,
    slug             varchar(140) NOT NULL,
    summary          varchar(300) NOT NULL DEFAULT '',
    description      text NOT NULL DEFAULT '',
    media_refs       jsonb NOT NULL DEFAULT '[]'::jsonb,
    service_mode     varchar(20) NOT NULL DEFAULT 'on_site',
    status           varchar(20) NOT NULL DEFAULT 'draft',
    visibility       varchar(20) NOT NULL DEFAULT 'public',
    locale           varchar(10) NOT NULL DEFAULT 'en',
    tags             jsonb NOT NULL DEFAULT '[]'::jsonb,
    price_from       numeric(14, 2) NULL,
    price_currency   varchar(3) NOT NULL DEFAULT '',
    extension_ref    jsonb NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT offerings_status_check
        CHECK (status IN ('draft', 'published', 'archived')),
    CONSTRAINT offerings_visibility_check
        CHECK (visibility IN ('public', 'unlisted', 'private')),
    CONSTRAINT offerings_service_mode_check
        CHECK (service_mode IN ('on_site', 'online', 'at_customer', 'hybrid')),
    CONSTRAINT offerings_org_slug_uniq UNIQUE (organization_id, slug)
);

CREATE INDEX IF NOT EXISTS offerings_org_idx      ON offerings (organization_id);
CREATE INDEX IF NOT EXISTS offerings_taxonomy_idx ON offerings (taxonomy_id);
CREATE INDEX IF NOT EXISTS offerings_public_idx
    ON offerings (status, visibility) WHERE status = 'published';

CREATE TABLE IF NOT EXISTS offering_variants (
    id            uuid PRIMARY KEY,
    offering_id   uuid NOT NULL REFERENCES offerings (id) ON DELETE CASCADE,
    name          varchar(160) NOT NULL,
    status        varchar(20) NOT NULL DEFAULT 'active',
    visibility    varchar(20) NOT NULL DEFAULT 'public',
    media_refs    jsonb NOT NULL DEFAULT '[]'::jsonb,
    extension_ref jsonb NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT offering_variants_visibility_check
        CHECK (visibility IN ('public', 'unlisted', 'private'))
);

CREATE INDEX IF NOT EXISTS offering_variants_offering_idx
    ON offering_variants (offering_id);
