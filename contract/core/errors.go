// Package core is the stable public surface of the core platform.
//
// Application code may import this package and the module packages it wires
// into main. It must not import platform internals directly; see PRD.md §8.4
// for the enforced boundary rules.
package core

import "errors"

// Sentinel errors. These are the only errors that cross module boundaries.
// The HTTP layer maps them to a single response envelope in one place.
var (
	ErrUnauthorized    = errors.New("unauthorized")
	ErrForbidden       = errors.New("forbidden")
	ErrNotFound        = errors.New("not found")
	ErrValidation      = errors.New("validation failed")
	ErrConflict        = errors.New("conflict")
	ErrRateLimited     = errors.New("rate limited")
	ErrNotImplemented  = errors.New("not implemented")
	ErrProviderFailure = errors.New("provider failure")
	// ErrInvalidState is returned when a requested status transition is not
	// allowed by the state machine that owns the record.
	ErrInvalidState = errors.New("invalid state transition")
	// ErrNotMember means the actor has no active membership in the tenant.
	ErrNotMember = errors.New("not a member")
)

// DomainError attaches a human-readable message to a sentinel without losing
// errors.Is. Handlers use it to explain a validation failure precisely.
type DomainError struct {
	// Kind is one of the sentinels above.
	Kind error
	// Message is safe to show to the caller.
	Message string
	// Field optionally names the offending input field.
	Field string
}

// Error implements error.
func (e *DomainError) Error() string {
	if e.Field != "" {
		return e.Message + " (" + e.Field + ")"
	}
	return e.Message
}

// Unwrap lets errors.Is match the sentinel.
func (e *DomainError) Unwrap() error { return e.Kind }

// Invalid builds a validation error for one field.
func Invalid(field, message string) error {
	return &DomainError{Kind: ErrValidation, Field: field, Message: message}
}

// StateError builds an invalid-transition error.
func StateError(message string) error {
	return &DomainError{Kind: ErrInvalidState, Message: message}
}
