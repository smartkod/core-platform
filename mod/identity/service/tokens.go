package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"core-platform/contract/core"
)

// Audience values keep access and refresh tokens from being interchangeable.
const (
	AudienceAccess = "access"
	AudienceMobile = "mobile"
	AudienceWeb    = "web"
)

// TokenConfig configures the token service.
type TokenConfig struct {
	AccessSecret       string
	RefreshSecret      string
	AccessTTL          time.Duration
	RefreshTTL         time.Duration
	RefreshTTLRemember time.Duration
	Issuer             string
}

// Claims is the access token payload.
//
// `ver` carries the user's token_version, so changing a password can invalidate
// every access token issued before it without a database lookup per request.
type Claims struct {
	Role  string   `json:"role"`
	Scope []string `json:"scopes,omitempty"`
	SID   string   `json:"sid"`
	Ver   int      `json:"ver"`
	jwt.RegisteredClaims
}

// Tokens issues and verifies JWTs.
//
// Access and refresh tokens use different secrets and different audiences: a
// leaked refresh secret must not let an attacker mint access tokens.
type Tokens struct {
	cfg TokenConfig
	now func() time.Time
}

// NewTokens builds the token service.
func NewTokens(cfg TokenConfig) (*Tokens, error) {
	if cfg.AccessSecret == "" {
		return nil, errors.New("identity: access token secret is required")
	}
	if cfg.RefreshSecret == "" {
		return nil, errors.New("identity: refresh token secret is required")
	}
	if cfg.AccessSecret == cfg.RefreshSecret {
		return nil, errors.New("identity: access and refresh secrets must differ")
	}
	if cfg.AccessTTL <= 0 {
		cfg.AccessTTL = 15 * time.Minute
	}
	if cfg.RefreshTTL <= 0 {
		cfg.RefreshTTL = 7 * 24 * time.Hour
	}
	if cfg.RefreshTTLRemember <= 0 {
		cfg.RefreshTTLRemember = 30 * 24 * time.Hour
	}
	if cfg.Issuer == "" {
		cfg.Issuer = "core-platform"
	}
	return &Tokens{cfg: cfg, now: func() time.Time { return time.Now().UTC() }}, nil
}

// IssuedAccess is a signed access token and its metadata.
type IssuedAccess struct {
	Token     string
	ExpiresAt time.Time
	JTI       string
}

// IssueAccess signs a short-lived access token.
func (t *Tokens) IssueAccess(userID, email, role string, scopes []string, version int, sessionID string) (IssuedAccess, error) {
	now := t.now()
	exp := now.Add(t.cfg.AccessTTL)
	jti := randomID()

	claims := Claims{
		Role:  role,
		Scope: scopes,
		SID:   sessionID,
		Ver:   version,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        jti,
			Issuer:    t.cfg.Issuer,
			Audience:  jwt.ClaimStrings{AudienceAccess},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	// The e-mail is informational only; it is not used for authorization.
	claims.Issuer = t.cfg.Issuer
	_ = email

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).
		SignedString([]byte(t.cfg.AccessSecret))
	if err != nil {
		return IssuedAccess{}, err
	}
	return IssuedAccess{Token: token, ExpiresAt: exp, JTI: jti}, nil
}

// ParseAccess verifies an access token and returns its claims.
func (t *Tokens) ParseAccess(raw string) (*Claims, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(raw, &claims, func(tok *jwt.Token) (any, error) {
		if _, ok := tok.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", tok.Header["alg"])
		}
		return []byte(t.cfg.AccessSecret), nil
	},
		jwt.WithAudience(AudienceAccess),
		jwt.WithIssuer(t.cfg.Issuer),
		jwt.WithValidMethods([]string{"HS256"}),
	)
	if err != nil {
		// A bad token is never an internal error; the caller maps 401.
		return nil, core.ErrUnauthorized
	}
	return &claims, nil
}

// RefreshValue is a refresh token as given to the client.
type RefreshValue struct {
	// Plain is the only time the raw value exists; it is never stored.
	Plain     string
	Hash      string
	ExpiresAt time.Time
}

// NewRefresh generates a refresh token.
//
// Refresh tokens are opaque random strings, not JWTs: they are validated by
// lookup, which is what makes rotation and reuse detection possible.
func (t *Tokens) NewRefresh(ttl time.Duration, remember bool) (RefreshValue, error) {
	if remember {
		ttl = t.cfg.RefreshTTLRemember
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return RefreshValue{}, err
	}
	plain := base64.RawURLEncoding.EncodeToString(buf)
	return RefreshValue{
		Plain:     plain,
		Hash:      HashToken(plain),
		ExpiresAt: t.now().Add(ttl),
	}, nil
}

// HashToken is the storage form of any opaque token.
//
// SHA-256 without a salt is correct here: the input is 256 bits of entropy, so
// there is nothing to brute-force, and an unsalted hash keeps lookups a single
// indexed query.
func HashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

// NewOneTime creates a single-use token such as a reset link.
func (t *Tokens) NewOneTime(ttl time.Duration) (RefreshValue, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return RefreshValue{}, err
	}
	plain := base64.RawURLEncoding.EncodeToString(buf)
	return RefreshValue{
		Plain:     plain,
		Hash:      HashToken(plain),
		ExpiresAt: t.now().Add(ttl),
	}, nil
}

func randomID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return base64.RawURLEncoding.EncodeToString([]byte(time.Now().String()))
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

// SessionID is a fresh identifier for a login. Every token in one family
// carries the same session id.
func SessionID() string { return randomID() }

// Used to keep the compile-time contract honest without exporting internals.
var _ = context.Background
