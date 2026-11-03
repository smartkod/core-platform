package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"core-platform/contract/core"
	"core-platform/mod/identity/model"
	"core-platform/mod/identity/repository"
)

// Mailer sends transactional e-mail. The identity module depends on this
// interface, not on SMTP, so tests can assert what would have been sent.
type Mailer interface {
	SendVerification(ctx context.Context, to, token string) error
	SendPasswordReset(ctx context.Context, to, token string) error
}

// nopMailer records nothing. Used when no mailer is configured, so a
// development environment is not blocked on an SMTP server.
type nopMailer struct{}

func (nopMailer) SendVerification(context.Context, string, string) error  { return nil }
func (nopMailer) SendPasswordReset(context.Context, string, string) error { return nil }

// NopMailer returns the no-op mailer.
func NopMailer() Mailer { return nopMailer{} }

// AuthConfig is the tunable policy.
type AuthConfig struct {
	// RefreshTTL applies to clients that did not ask to be remembered.
	RefreshTTL time.Duration
	// RefreshTTLRemember applies to web clients with "remember me".
	RefreshTTLRemember time.Duration
	// VerifyTTL bounds e-mail verification links.
	VerifyTTL time.Duration
	// ResetTTL bounds password reset links.
	ResetTTL time.Duration
}

// NewAuthConfig fills in defaults.
func NewAuthConfig() AuthConfig {
	return AuthConfig{
		RefreshTTL:         7 * 24 * time.Hour,
		RefreshTTLRemember: 30 * 24 * time.Hour,
		VerifyTTL:          24 * time.Hour,
		ResetTTL:           1 * time.Hour,
	}
}

// Auth is the identity application service.
type Auth struct {
	users  *repository.Users
	tokens *repository.Tokens
	tok    *Tokens
	pw     *Password
	mailer Mailer
	cfg    AuthConfig
	now    func() time.Time
}

// NewAuth wires the service.
func NewAuth(
	users *repository.Users,
	tokens *repository.Tokens,
	tok *Tokens,
	pw *Password,
	mailer Mailer,
	cfg AuthConfig,
) *Auth {
	if mailer == nil {
		mailer = nopMailer{}
	}
	return &Auth{
		users: users, tokens: tokens, tok: tok, pw: pw,
		mailer: mailer, cfg: cfg,
		now: func() time.Time { return time.Now().UTC() },
	}
}

// Session is what a successful login or refresh returns.
type Session struct {
	Access         IssuedAccess
	Refresh        string
	RefreshExpires time.Time
	User           *model.User
}

// Register creates an account.
//
// Registration deliberately does not reveal whether an e-mail is taken: the
// transport layer returns the same message either way, and this method returns
// ErrConflict only so internal callers can branch.
func (a *Auth) Register(ctx context.Context, in RegisterInput) (*model.User, error) {
	email := normalizeEmail(in.Email)
	if email == "" {
		return nil, core.ErrValidation
	}
	if err := a.pw.Validate(in.Password); err != nil {
		return nil, err
	}

	if _, err := a.users.ByEmail(ctx, email); err == nil {
		return nil, core.ErrConflict
	} else if !errors.Is(err, core.ErrNotFound) {
		return nil, err
	}

	hash, err := a.pw.Hash(in.Password)
	if err != nil {
		return nil, err
	}

	now := a.now()
	u := &model.User{
		ID:           newUUID(),
		Email:        email,
		PasswordHash: hash,
		Name:         strings.TrimSpace(in.Name),
		Locale:       defaultString(in.Locale, "en"),
		Role:         model.RoleUser,
		Status:       model.StatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := a.users.Create(ctx, u); err != nil {
		return nil, err
	}

	if err := a.sendOneTime(ctx, u, model.PurposeVerifyEmail, a.cfg.VerifyTTL); err != nil {
		// A mail failure must not fail the registration: the account exists and
		// the user can request another link.
		_ = err
	}
	return u, nil
}

// Login verifies credentials and opens a session.
func (a *Auth) Login(ctx context.Context, in LoginInput) (*Session, error) {
	u, err := a.users.ByEmail(ctx, normalizeEmail(in.Email))
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			// Run the hasher anyway so a missing account and a wrong password
			// take the same time; otherwise login doubles as an account oracle.
			_, _ = a.pw.Hash(in.Password)
			return nil, core.ErrUnauthorized
		}
		return nil, err
	}
	if err := a.pw.Verify(u.PasswordHash, in.Password); err != nil {
		return nil, core.ErrUnauthorized
	}
	if u.Status != model.StatusActive {
		return nil, core.ErrForbidden
	}

	session, err := a.openSession(ctx, u, in)
	if err != nil {
		return nil, err
	}

	now := a.now()
	u.LastLoginAt = &now
	if err := a.users.Update(ctx, u); err != nil {
		return nil, err
	}
	return session, nil
}

func (a *Auth) openSession(ctx context.Context, u *model.User, in LoginInput) (*Session, error) {
	ttl := a.cfg.RefreshTTL
	if in.Remember {
		ttl = a.cfg.RefreshTTLRemember
	}

	familyID := newUUID()
	refresh, err := a.tok.NewRefresh(ttl, in.Remember)
	if err != nil {
		return nil, err
	}

	row := &model.RefreshToken{
		ID:        newUUID(),
		UserID:    u.ID,
		TokenHash: refresh.Hash,
		FamilyID:  familyID,
		Device:    truncate(in.Device, 120),
		IP:        truncate(in.IP, 64),
		Audience:  defaultString(in.Audience, AudienceMobile),
		ExpiresAt: refresh.ExpiresAt,
		CreatedAt: a.now(),
	}
	if err := a.tokens.Create(ctx, row); err != nil {
		return nil, err
	}

	access, err := a.tok.IssueAccess(u.ID, u.Email, u.Role, nil, u.TokenVersion, familyID)
	if err != nil {
		return nil, err
	}

	return &Session{
		Access:         access,
		Refresh:        refresh.Plain,
		RefreshExpires: refresh.ExpiresAt,
		User:           u,
	}, nil
}

// Refresh rotates a refresh token.
//
// Rotation is mandatory: the presented token is revoked and a new one issued in
// the same family. If a token that was already revoked is presented again, the
// entire family is revoked, because that pattern means the token leaked
// (PRD.md §6.1).
func (a *Auth) Refresh(ctx context.Context, in RefreshInput) (*Session, error) {
	hash := HashToken(in.RefreshToken)
	row, err := a.tokens.ByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, core.ErrUnauthorized
	}

	if row.RevokedAt != nil {
		// Reuse of a revoked token: assume theft and cut the whole family.
		_ = a.tokens.RevokeFamily(ctx, row.FamilyID)
		return nil, core.ErrUnauthorized
	}
	if a.now().After(row.ExpiresAt) {
		return nil, core.ErrUnauthorized
	}

	u, err := a.users.ByID(ctx, row.UserID)
	if err != nil {
		return nil, core.ErrUnauthorized
	}
	if u.Status != model.StatusActive {
		return nil, core.ErrForbidden
	}

	remember := row.Audience == AudienceWeb
	ttl := a.cfg.RefreshTTL
	if remember {
		ttl = a.cfg.RefreshTTLRemember
	}

	next, err := a.tok.NewRefresh(ttl, remember)
	if err != nil {
		return nil, err
	}
	nextID := newUUID()

	newRow := &model.RefreshToken{
		ID:        nextID,
		UserID:    u.ID,
		TokenHash: next.Hash,
		FamilyID:  row.FamilyID,
		Device:    row.Device,
		IP:        row.IP,
		Audience:  row.Audience,
		ExpiresAt: next.ExpiresAt,
		CreatedAt: a.now(),
	}
	if err := a.tokens.Create(ctx, newRow); err != nil {
		return nil, err
	}
	if err := a.tokens.Revoke(ctx, row.ID, nextID); err != nil {
		return nil, err
	}

	access, err := a.tok.IssueAccess(u.ID, u.Email, u.Role, nil, u.TokenVersion, row.FamilyID)
	if err != nil {
		return nil, err
	}

	return &Session{
		Access:         access,
		Refresh:        next.Plain,
		RefreshExpires: next.ExpiresAt,
		User:           u,
	}, nil
}

// Logout ends one session by revoking the whole family.
func (a *Auth) Logout(ctx context.Context, refreshToken string) error {
	row, err := a.tokens.ByHash(ctx, HashToken(refreshToken))
	if err != nil {
		return err
	}
	if row == nil {
		// Already gone: logout is idempotent and must not leak whether the
		// token existed.
		return nil
	}
	return a.tokens.RevokeFamily(ctx, row.FamilyID)
}

// LogoutEverywhere revokes every session and invalidates all access tokens.
func (a *Auth) LogoutEverywhere(ctx context.Context, userID string) error {
	if err := a.tokens.RevokeAllForUser(ctx, userID); err != nil {
		return err
	}
	return a.users.BumpTokenVersion(ctx, userID)
}

// ForgotPassword issues a reset link. It never reveals whether the e-mail
// exists.
func (a *Auth) ForgotPassword(ctx context.Context, email string) error {
	u, err := a.users.ByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return nil
		}
		return err
	}
	return a.sendOneTime(ctx, u, model.PurposeResetPassword, a.cfg.ResetTTL)
}

// ResetPassword consumes a reset token and rotates every credential.
func (a *Auth) ResetPassword(ctx context.Context, token, newPassword string) error {
	if err := a.pw.Validate(newPassword); err != nil {
		return err
	}

	row, err := a.tokens.ConsumeOneTime(ctx, HashToken(token), model.PurposeResetPassword)
	if err != nil {
		return err
	}
	if row == nil {
		return core.ErrValidation
	}

	u, err := a.users.ByID(ctx, row.UserID)
	if err != nil {
		return core.ErrValidation
	}

	hash, err := a.pw.Hash(newPassword)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	if err := a.users.Update(ctx, u); err != nil {
		return err
	}

	// A password change must log everyone out, including the attacker who
	// forced the reset.
	if err := a.tokens.RevokeAllForUser(ctx, u.ID); err != nil {
		return err
	}
	return a.users.BumpTokenVersion(ctx, u.ID)
}

// VerifyEmail consumes a verification token.
func (a *Auth) VerifyEmail(ctx context.Context, token string) error {
	row, err := a.tokens.ConsumeOneTime(ctx, HashToken(token), model.PurposeVerifyEmail)
	if err != nil {
		return err
	}
	if row == nil {
		return core.ErrValidation
	}

	u, err := a.users.ByID(ctx, row.UserID)
	if err != nil {
		return core.ErrValidation
	}
	now := a.now()
	u.EmailVerifiedAt = &now
	return a.users.Update(ctx, u)
}

// DeleteSelf soft-deletes an account and terminates every session it owns.
//
// Deleting is soft because audit trails, orders and files reference the user
// row; a hard delete would either cascade surprise data away or fail on
// foreign keys. A retention job removes the row later.
func (a *Auth) DeleteSelf(ctx context.Context, userID string) error {
	if _, err := a.users.ByID(ctx, userID); err != nil {
		return err
	}
	if err := a.tokens.RevokeAllForUser(ctx, userID); err != nil {
		return err
	}
	return a.users.SoftDelete(ctx, userID)
}

func (a *Auth) sendOneTime(ctx context.Context, u *model.User, purpose string, ttl time.Duration) error {
	val, err := a.tok.NewOneTime(ttl)
	if err != nil {
		return err
	}
	row := &model.OneTimeToken{
		ID:        newUUID(),
		UserID:    u.ID,
		Purpose:   purpose,
		TokenHash: val.Hash,
		ExpiresAt: val.ExpiresAt,
		CreatedAt: a.now(),
	}
	if err := a.tokens.CreateOneTime(ctx, row); err != nil {
		return err
	}

	switch purpose {
	case model.PurposeVerifyEmail:
		return a.mailer.SendVerification(ctx, u.Email, val.Plain)
	case model.PurposeResetPassword:
		return a.mailer.SendPasswordReset(ctx, u.Email, val.Plain)
	}
	return nil
}

// RegisterInput is the service-level registration request.
type RegisterInput struct {
	Email    string
	Password string
	Name     string
	Locale   string
}

// LoginInput is the service-level login request.
type LoginInput struct {
	Email    string
	Password string
	Device   string
	IP       string
	Audience string
	Remember bool
}

// RefreshInput carries the presented refresh token.
type RefreshInput struct {
	RefreshToken string
}

func normalizeEmail(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func defaultString(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
