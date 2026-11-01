package service

import "core-platform/platform/uuid"

// newUUID is the single place the identity module mints identifiers, so a test
// could substitute a deterministic generator if it ever needed to.
func newUUID() string { return uuid.New() }
