-- 0004_authz_init: roles, permissions and scoped assignments.
--
-- The whole point of this schema is that authority is never global by accident.
-- A role is a set of permissions, and an assignment binds one role to one user
-- inside one scope (platform, organization, branch or self). A check that finds
-- no active assignment is denied.

CREATE TABLE IF NOT EXISTS roles (
    key         varchar(80) PRIMARY KEY,
    name        varchar(120) NOT NULL,
    description text NOT NULL DEFAULT '',
    system      boolean NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS permissions (
    key         varchar(120) PRIMARY KEY,
    description text NOT NULL DEFAULT '',
    scope       varchar(20) NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT permissions_scope_check
        CHECK (scope IN ('platform', 'organization', 'branch', 'self'))
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_key       varchar(80) NOT NULL REFERENCES roles (key) ON DELETE CASCADE,
    permission_key varchar(120) NOT NULL REFERENCES permissions (key) ON DELETE CASCADE,
    PRIMARY KEY (role_key, permission_key)
);

CREATE TABLE IF NOT EXISTS role_assignments (
    id         uuid PRIMARY KEY,
    user_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_key   varchar(80) NOT NULL REFERENCES roles (key) ON DELETE RESTRICT,
    scope      varchar(20) NOT NULL,
    scope_id   varchar(64) NOT NULL DEFAULT '',
    granted_by uuid NULL,
    expires_at timestamptz NULL,
    revoked_at timestamptz NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT role_assignments_scope_check
        CHECK (scope IN ('platform', 'organization', 'branch', 'self'))
);

CREATE INDEX IF NOT EXISTS role_assignments_user_idx
    ON role_assignments (user_id) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS role_assignments_scope_idx
    ON role_assignments (scope, scope_id) WHERE revoked_at IS NULL;
-- One active grant per (user, role, scope, scope_id) keeps a double-click from
-- creating duplicate rows that can never be told apart in the audit trail.
CREATE UNIQUE INDEX IF NOT EXISTS role_assignments_active_uniq
    ON role_assignments (user_id, role_key, scope, scope_id)
    WHERE revoked_at IS NULL;

-- Seed the core permission vocabulary. An extension registers its own keys at
-- runtime, so this list stays short and belongs to the core.
INSERT INTO permissions (key, description, scope) VALUES
    ('platform.admin',          'Administer the platform itself',            'platform'),
    ('moderation.review',       'Review reported content',                   'platform'),
    ('organization.read',       'Read an organization profile',              'organization'),
    ('organization.manage',     'Edit an organization profile',              'organization'),
    ('organization.publish',    'Publish or suspend an organization',        'platform'),
    ('org.branch.manage',       'Manage branches of an organization',        'organization'),
    ('org.member.manage',       'Invite, remove and re-role members',        'organization'),
    ('catalog.read',            'Read catalogue entries',                    'organization'),
    ('catalog.manage',          'Create and edit offerings',                 'organization'),
    ('catalog.publish',         'Publish or archive offerings',              'organization'),
    ('taxonomy.manage',         'Manage the central category tree',          'platform'),
    ('review.respond',          'Respond to a review as the business',       'organization')
ON CONFLICT (key) DO NOTHING;

-- System roles. They are marked system so a delete attempt can be rejected;
-- an application may still add its own roles alongside them.
INSERT INTO roles (key, name, description, system) VALUES
    ('platform_admin', 'Platform administrator',
        'Full control over the platform and every organization', true),
    ('moderator', 'Moderator',
        'Reviews content and organizations without changing accounts', true),
    ('owner', 'Organization owner',
        'Full control inside one organization, including ownership transfer', true),
    ('manager', 'Organization manager',
        'Runs day-to-day organization and catalogue work', true),
    ('staff', 'Organization staff',
        'Reads and edits catalogue entries in assigned branches', true),
    ('member', 'Member',
        'Belongs to the organization without management rights', true)
ON CONFLICT (key) DO NOTHING;

INSERT INTO role_permissions (role_key, permission_key) VALUES
    ('platform_admin', 'platform.admin'),
    ('platform_admin', 'moderation.review'),
    ('platform_admin', 'organization.read'),
    ('platform_admin', 'organization.manage'),
    ('platform_admin', 'organization.publish'),
    ('platform_admin', 'org.branch.manage'),
    ('platform_admin', 'org.member.manage'),
    ('platform_admin', 'catalog.read'),
    ('platform_admin', 'catalog.manage'),
    ('platform_admin', 'catalog.publish'),
    ('platform_admin', 'taxonomy.manage'),
    ('platform_admin', 'review.respond'),

    ('moderator', 'moderation.review'),
    ('moderator', 'organization.read'),

    ('owner', 'organization.read'),
    ('owner', 'organization.manage'),
    ('owner', 'org.branch.manage'),
    ('owner', 'org.member.manage'),
    ('owner', 'catalog.read'),
    ('owner', 'catalog.manage'),
    ('owner', 'catalog.publish'),
    ('owner', 'review.respond'),

    ('manager', 'organization.read'),
    ('manager', 'organization.manage'),
    ('manager', 'org.branch.manage'),
    ('manager', 'catalog.read'),
    ('manager', 'catalog.manage'),
    ('manager', 'catalog.publish'),
    ('manager', 'review.respond'),

    ('staff', 'organization.read'),
    ('staff', 'catalog.read'),
    ('staff', 'catalog.manage'),

    ('member', 'organization.read'),
    ('member', 'catalog.read')
ON CONFLICT DO NOTHING;
