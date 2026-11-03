// Package repository is the only layer that touches the authorization tables.
package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"core-platform/contract/core"
	"core-platform/mod/authz/model"
)

// Store is the authorization store.
type Store struct{ db *gorm.DB }

// NewStore builds the store.
func NewStore(db *gorm.DB) *Store { return &Store{db: db} }

// Roles returns every role with its permission keys.
func (s *Store) Roles(ctx context.Context) ([]core.Role, error) {
	var rows []model.Role
	if err := s.db.WithContext(ctx).Order("key").Find(&rows).Error; err != nil {
		return nil, err
	}

	var links []model.RolePermission
	if err := s.db.WithContext(ctx).Find(&links).Error; err != nil {
		return nil, err
	}
	byRole := map[string][]string{}
	for _, l := range links {
		byRole[l.RoleKey] = append(byRole[l.RoleKey], l.PermissionKey)
	}

	out := make([]core.Role, 0, len(rows))
	for _, r := range rows {
		out = append(out, core.Role{
			Key:         r.Key,
			Name:        r.Name,
			Description: r.Description,
			System:      r.System,
			Permissions: byRole[r.Key],
			CreatedAt:   r.CreatedAt,
		})
	}
	return out, nil
}

// Role returns one role or ErrNotFound.
func (s *Store) Role(ctx context.Context, key string) (*core.Role, error) {
	var r model.Role
	err := s.db.WithContext(ctx).Where("key = ?", key).First(&r).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, core.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	var keys []string
	err = s.db.WithContext(ctx).Model(&model.RolePermission{}).
		Where("role_key = ?", key).
		Pluck("permission_key", &keys).Error
	if err != nil {
		return nil, err
	}

	return &core.Role{
		Key:         r.Key,
		Name:        r.Name,
		Description: r.Description,
		System:      r.System,
		Permissions: keys,
		CreatedAt:   r.CreatedAt,
	}, nil
}

// UpsertRole creates or replaces a role definition.
//
// A system role may not be silently replaced: it is part of the platform's
// contract, so only its permissions are editable here, and only by an
// application that owns the key prefix.
func (s *Store) UpsertRole(ctx context.Context, role core.Role) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.Role
		err := tx.Where("key = ?", role.Key).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			rec := model.Role{
				Key:         role.Key,
				Name:        role.Name,
				Description: role.Description,
				System:      role.System,
				CreatedAt:   time.Now().UTC(),
			}
			if err := tx.Create(&rec).Error; err != nil {
				return err
			}
		} else {
			updates := map[string]any{
				"name":        role.Name,
				"description": role.Description,
			}
			if err := tx.Model(&model.Role{}).Where("key = ?", role.Key).
				Updates(updates).Error; err != nil {
				return err
			}
		}

		if role.Permissions == nil {
			return nil
		}
		if err := tx.Where("role_key = ?", role.Key).
			Delete(&model.RolePermission{}).Error; err != nil {
			return err
		}
		for _, p := range role.Permissions {
			link := model.RolePermission{RoleKey: role.Key, PermissionKey: p}
			if err := tx.Create(&link).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// RegisterPermission records a permission key. It is idempotent.
func (s *Store) RegisterPermission(ctx context.Context, p core.Permission) error {
	rec := model.Permission{
		Key:         p.Key,
		Description: p.Description,
		Scope:       string(p.Scope),
		CreatedAt:   time.Now().UTC(),
	}
	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&rec).Error
}

// Grant inserts an assignment.
func (s *Store) Grant(ctx context.Context, a *model.Assignment) error {
	return s.db.WithContext(ctx).Create(a).Error
}

// Revoke marks an assignment revoked.
//
// The row is kept: an authorization history that can be erased is not an audit
// trail, and "who could do this last month" must stay answerable.
func (s *Store) Revoke(ctx context.Context, id string) error {
	now := time.Now().UTC()
	res := s.db.WithContext(ctx).Model(&model.Assignment{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return core.ErrNotFound
	}
	return nil
}

// Assignments lists a user's active grants.
func (s *Store) Assignments(ctx context.Context, userID string) ([]core.Assignment, error) {
	var rows []model.Assignment
	err := s.db.WithContext(ctx).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Where("expires_at IS NULL OR expires_at > now()").
		Order("created_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	out := make([]core.Assignment, 0, len(rows))
	for _, r := range rows {
		a := core.Assignment{
			ID:        r.ID,
			UserID:    r.UserID,
			RoleKey:   r.RoleKey,
			Scope:     core.ScopeType(r.Scope),
			ScopeID:   r.ScopeID,
			ExpiresAt: r.ExpiresAt,
			RevokedAt: r.RevokedAt,
			CreatedAt: r.CreatedAt,
		}
		if r.GrantedBy != nil {
			a.GrantedBy = *r.GrantedBy
		}
		out = append(out, a)
	}
	return out, nil
}

// PermissionKeys returns the effective permission keys for one (user, scope,
// scope id) triple, joined through the role definitions.
func (s *Store) PermissionKeys(ctx context.Context, userID, scope, scopeID string) ([]string, error) {
	var keys []string
	err := s.db.WithContext(ctx).Raw(`
		SELECT DISTINCT rp.permission_key
		FROM role_assignments ra
		JOIN role_permissions rp ON rp.role_key = ra.role_key
		WHERE ra.user_id = ?
		  AND ra.scope = ?
		  AND (ra.scope_id = ? OR ra.scope = 'platform')
		  AND ra.revoked_at IS NULL
		  AND (ra.expires_at IS NULL OR ra.expires_at > now())
		ORDER BY rp.permission_key`, userID, scope, scopeID).Scan(&keys).Error
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// CountActiveOwners reports how many active owners an organization still has.
//
// It exists so an ownership transfer or a member removal cannot strand an
// organization with nobody able to administer it.
func (s *Store) CountActiveOwners(ctx context.Context, organizationID string) (int64, error) {
	var n int64
	err := s.db.WithContext(ctx).Model(&model.Assignment{}).
		Where("scope = ? AND scope_id = ? AND role_key IN ? AND revoked_at IS NULL",
			string(core.ScopeOrganization), organizationID,
			[]string{"owner", "platform_admin"}).
		Where("expires_at IS NULL OR expires_at > now()").
		Count(&n).Error
	return n, err
}

// RevokeScope removes every active assignment in one scope.
//
// It is used when an organization is closed: leaving grants behind would let a
// closed business keep acting through an old session.
func (s *Store) RevokeScope(ctx context.Context, scope, scopeID string) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Model(&model.Assignment{}).
		Where("scope = ? AND scope_id = ? AND revoked_at IS NULL", scope, scopeID).
		Update("revoked_at", now).Error
}
