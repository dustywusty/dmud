package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestIdentityClassifyAndStore(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	// IDENTITY| is captured silently (not shown in OUTPUT).
	r := classify("IDENTITY|abc-123")
	if r.identity != "abc-123" {
		t.Fatalf("identity = %q, want abc-123", r.identity)
	}
	if r.toMain != "" || r.toChat != "" {
		t.Errorf("IDENTITY should be silent, got %+v", r)
	}

	// Per-server round-trip on disk.
	if err := saveIdentity("myhost:8080", "abc-123"); err != nil {
		t.Fatal(err)
	}
	_ = saveIdentity("other:3333", "xyz")
	if got := loadIdentity("myhost:8080"); got != "abc-123" {
		t.Errorf("loadIdentity(myhost) = %q, want abc-123", got)
	}
	if got := loadIdentity("unknown"); got != "" {
		t.Errorf("unknown server should have no saved id, got %q", got)
	}

	// The model captures the id from a chunk so it can be replayed next launch.
	m := newModel(newConn(transportWS, "myhost:8080"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 30}, chunkMsg{raw: "IDENTITY|fresh-id"})
	if m.identity != "fresh-id" {
		t.Errorf("model should capture identity from a chunk, got %q", m.identity)
	}
}
