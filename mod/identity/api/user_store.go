package api

import (
	"context"

	"core-platform/contract/core"
	"core-platform/mod/identity/model"
)

// GetUser implements core.UserStore.
func (i *Identity) GetUser(ctx context.Context, id string) (*core.User, error) {
	u, err := i.users.ByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return publicUser(u), nil
}

// FindUserByEmail implements core.UserStore.
func (i *Identity) FindUserByEmail(ctx context.Context, email string) (*core.User, error) {
	u, err := i.users.ByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return publicUser(u), nil
}

// publicUser projects a stored account onto the narrow public type.
//
// The password hash, token version and last-login stamp never leave the
// identity module: an engine has no reason to see them, and leaking them widens
// the blast radius of a mistake in a downstream handler.
func publicUser(u *model.User) *core.User {
	if u == nil {
		return nil
	}
	return &core.User{
		ID:              u.ID,
		Email:           u.Email,
		Phone:           u.Phone,
		Name:            u.Name,
		Locale:          u.Locale,
		AvatarURL:       u.AvatarURL,
		Role:            u.Role,
		Status:          u.Status,
		EmailVerifiedAt: u.EmailVerifiedAt,
		PhoneVerifiedAt: u.PhoneVerifiedAt,
		ClosedAt:        u.ClosedAt,
		CreatedAt:       u.CreatedAt,
	}
}
