package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestJoinMessageSpacing(t *testing.T) {
	if got := joinMessage("first", "second", true); got != "first\n\nsecond" {
		t.Errorf("roomy join = %q, want blank line between", got)
	}
	if got := joinMessage("first", "second", false); got != "first\nsecond" {
		t.Errorf("compact join = %q, want single newline", got)
	}
	if got := joinMessage("", "first", true); got != "first" {
		t.Errorf("first message should have no leading separator, got %q", got)
	}
}

func TestSpacingToggle(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 30})

	// Roomy by default: a blank line separates messages.
	m.appendMain("alpha")
	m.appendMain("beta")
	if !strings.Contains(m.mainRaw, "alpha\n\nbeta") {
		t.Errorf("default spacing should blank-separate messages: %q", m.mainRaw)
	}

	// /spacing switches to compact for subsequent messages.
	m.runSlash("/spacing")
	m.appendMain("gamma")
	m.appendMain("delta")
	if !strings.Contains(m.mainRaw, "gamma\ndelta") || strings.Contains(m.mainRaw, "gamma\n\ndelta") {
		t.Errorf("compact should single-space new messages: %q", m.mainRaw)
	}

	// COMMS honors the same setting.
	if m.comms.roomy {
		t.Error("toggling spacing should also make COMMS compact")
	}
}
