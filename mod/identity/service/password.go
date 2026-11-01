package service

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"

	"core-platform/contract/core"
)

// Password holds the hashing policy.
type Password struct {
	minLength int
	maxLength int
}

// NewPassword builds a hasher with the configured policy.
//
// The upper bound exists because Argon2 cost grows with input and because a
// multi-megabyte "password" is an easy denial-of-service.
func NewPassword(minLength int) *Password {
	if minLength <= 0 {
		minLength = 10
	}
	return &Password{minLength: minLength, maxLength: 72}
}

// Validate enforces the policy and returns the canonical sentinel so the HTTP
// layer maps it to 422 without the handler knowing the rule.
func (p *Password) Validate(plain string) error {
	if len(plain) < p.minLength {
		return fmt.Errorf("%w: password must be at least %d characters", core.ErrValidation, p.minLength)
	}
	if len(plain) > p.maxLength {
		return fmt.Errorf("%w: password must be at most %d characters", core.ErrValidation, p.maxLength)
	}
	return nil
}

// Argon2id parameters.
//
// These follow the OWASP minimum (19 MiB, t=2, p=1) and are stored inside the
// hash string, so raising them later does not invalidate existing passwords.
const (
	argonTime    uint32 = 2
	argonMemory  uint32 = 19 * 1024
	argonThreads uint8  = 1
	argonKeyLen  uint32 = 32
	argonSaltLen        = 16
)

// Hash derives an encoded hash: $argon2id$v=19$m=...,t=...,p=...$salt$key
func (p *Password) Hash(plain string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(plain), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// Verify compares a plain password against an encoded hash in constant time.
//
// It returns nil only on a match; any parse failure is also a non-match, never
// an internal error, so a malformed stored hash cannot be distinguished from a
// wrong password by a caller.
func (p *Password) Verify(encoded, plain string) error {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return core.ErrUnauthorized
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return core.ErrUnauthorized
	}

	var memory, iter uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iter, &threads); err != nil {
		return core.ErrUnauthorized
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return core.ErrUnauthorized
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return core.ErrUnauthorized
	}

	got := argon2.IDKey([]byte(plain), salt, iter, memory, threads, uint32(len(want)))
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return core.ErrUnauthorized
	}
	return nil
}

// ErrPasswordPolicy is kept out of the sentinel set on purpose; callers wrap
// core.ErrValidation with the human-readable reason instead.
var ErrPasswordPolicy = errors.New("password policy")
