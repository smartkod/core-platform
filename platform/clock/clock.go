// Package clock provides an injectable time source.
//
// Scheduled behaviour (order expiry, subscription end) is tested by advancing
// a fake clock instead of sleeping, so tests are deterministic.
package clock

import "time"

// Real is the production clock.
type Real struct{}

// Now returns the current wall clock time in UTC.
func (Real) Now() time.Time { return time.Now().UTC() }

// Fake is a controllable clock for tests.
type Fake struct{ t time.Time }

// NewFake returns a fake clock pinned to t.
func NewFake(t time.Time) *Fake { return &Fake{t: t} }

// Now returns the pinned time.
func (f *Fake) Now() time.Time { return f.t }

// Advance moves the clock forward by d.
func (f *Fake) Advance(d time.Duration) { f.t = f.t.Add(d) }
