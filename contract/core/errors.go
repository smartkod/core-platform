// Package core is the stable public surface of the core platform.
//
// Application code may import this package and the module packages it wires
// into main. It must not import platform internals directly; see PRD.md 4.1
// for the enforced boundary rules.
package core

import "errors"

// Sentinel errors. These are the only errors that cross module boundaries.
// The HTTP layer maps them to a single response envelope in one place.
var (
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden")
	ErrNotFound            = errors.New("not found")
	ErrValidation          = errors.New("validation failed")
	ErrConflict            = errors.New("conflict")
	ErrRateLimited         = errors.New("rate limited")
	ErrNotImplemented      = errors.New("not implemented")
	ErrProviderFailure     = errors.New("provider failure")
	ErrInsufficientBalance = errors.New("insufficient balance")
	ErrEntitlementRequired = errors.New("entitlement required")
)
