package module

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"core-platform/contract/core"
)

// stub is a module that records lifecycle calls so ordering can be asserted
// from the outside rather than from internals.
type stub struct {
	name    string
	deps    []string
	log     *[]string
	startOK bool
}

func newStub(name string, deps []string, log *[]string) *stub {
	return &stub{name: name, deps: deps, log: log, startOK: true}
}

func (s *stub) Name() string           { return s.name }
func (s *stub) Version() string        { return "1.0.0" }
func (s *stub) Dependencies() []string { return s.deps }
func (s *stub) Migrate(context.Context, *gorm.DB) error {
	return nil
}
func (s *stub) RegisterRoutes(*echo.Group, core.Deps) {}
func (s *stub) Start(context.Context, core.Deps) error {
	if !s.startOK {
		return errors.New("start failed")
	}
	*s.log = append(*s.log, "start:"+s.name)
	return nil
}
func (s *stub) Stop(context.Context) error {
	*s.log = append(*s.log, "stop:"+s.name)
	return nil
}

func TestDependencyOrder(t *testing.T) {
	var log []string
	// Declared out of order on purpose: c depends on b, b depends on a.
	mods := []core.Module{
		newStub("c", []string{"b"}, &log),
		newStub("a", nil, &log),
		newStub("b", []string{"a"}, &log),
	}

	m, err := New(mods, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"a", "b", "c"}
	if got := m.Order(); !reflect.DeepEqual(got, want) {
		t.Fatalf("order = %v, want %v", got, want)
	}

	if err := m.Start(context.Background(), core.Deps{}); err != nil {
		t.Fatalf("start: %v", err)
	}
	wantCalls := []string{"start:a", "start:b", "start:c"}
	if !reflect.DeepEqual(log, wantCalls) {
		t.Fatalf("start order = %v, want %v", log, wantCalls)
	}
}

func TestUnknownDependencyFailsClosed(t *testing.T) {
	var log []string
	mods := []core.Module{newStub("a", []string{"missing"}, &log)}

	if _, err := New(mods, nil); err == nil {
		t.Fatal("expected error for unknown dependency")
	}
}

func TestCycleDetected(t *testing.T) {
	var log []string
	mods := []core.Module{
		newStub("a", []string{"c"}, &log),
		newStub("b", []string{"a"}, &log),
		newStub("c", []string{"b"}, &log),
	}

	if _, err := New(mods, nil); err == nil {
		t.Fatal("expected error for dependency cycle")
	}
}

func TestDuplicateNameRejected(t *testing.T) {
	var log []string
	mods := []core.Module{
		newStub("a", nil, &log),
		newStub("a", nil, &log),
	}

	if _, err := New(mods, nil); err == nil {
		t.Fatal("expected error for duplicate module name")
	}
}

func TestStartFailureRollsBackInReverse(t *testing.T) {
	var log []string
	bad := newStub("bad", []string{"a"}, &log)
	bad.startOK = false

	mods := []core.Module{
		newStub("a", nil, &log),
		bad,
		newStub("c", []string{"bad"}, &log),
	}

	m, err := New(mods, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := m.Start(context.Background(), core.Deps{}); err == nil {
		t.Fatal("expected start error")
	}

	// a must be stopped, c must never have started.
	want := []string{"start:a", "stop:a"}
	if !reflect.DeepEqual(log, want) {
		t.Fatalf("calls = %v, want %v", log, want)
	}
}

func TestDisabledModuleIsSkipped(t *testing.T) {
	var log []string
	mods := []core.Module{
		newStub("a", nil, &log),
		newStub("b", []string{"a"}, &log),
	}

	m, err := New(mods, map[string]bool{"a": true, "b": false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := m.Start(context.Background(), core.Deps{}); err != nil {
		t.Fatalf("start: %v", err)
	}

	if len(log) != 1 || log[0] != "start:a" {
		t.Fatalf("calls = %v, want only start:a", log)
	}
}
