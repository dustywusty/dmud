package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHistoryNavigation(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel(newConn(transportWS, "x"), nil)

	m.submit("look")
	m.submit("north")
	m.submit("north") // consecutive duplicate should not be stored twice
	if len(m.history) != 2 {
		t.Fatalf("history = %v, want [look north] (deduped)", m.history)
	}

	// Type a draft, then browse up through history and back down.
	m.in.SetValue("dr")
	up := tea.KeyMsg{Type: tea.KeyUp}
	down := tea.KeyMsg{Type: tea.KeyDown}

	m = drive(m, up)
	if m.in.Value() != "north" {
		t.Errorf("first up = %q, want north", m.in.Value())
	}
	m = drive(m, up)
	if m.in.Value() != "look" {
		t.Errorf("second up = %q, want look", m.in.Value())
	}
	m = drive(m, down)
	if m.in.Value() != "north" {
		t.Errorf("down = %q, want north", m.in.Value())
	}
	m = drive(m, down)
	if m.in.Value() != "dr" {
		t.Errorf("down past end should restore the draft, got %q", m.in.Value())
	}
}

func TestHistoryPersistence(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	appendHistoryCmd("look")() // execute the returned command
	appendHistoryCmd("north")()
	if c := appendHistoryCmd("login secret-uuid"); c != nil {
		t.Error("login commands must not be persisted to disk")
	}

	got := loadHistory()
	if len(got) != 2 || got[0] != "look" || got[1] != "north" {
		t.Errorf("loadHistory = %v, want [look north]", got)
	}
}
