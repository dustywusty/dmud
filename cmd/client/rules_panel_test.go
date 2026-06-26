package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestMapKeyToggle(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 30}, tea.KeyMsg{Type: tea.KeyCtrlG})
	if !m.fullMap {
		t.Error("ctrl+g should open the full-screen map")
	}
	// Esc closes it.
	m = drive(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.fullMap {
		t.Error("esc should close the full-screen map")
	}
	// ctrl+g still toggles it off as well.
	m = drive(m, tea.KeyMsg{Type: tea.KeyCtrlG}, tea.KeyMsg{Type: tea.KeyCtrlG})
	if m.fullMap {
		t.Error("ctrl+g should toggle the map off")
	}
}

func TestRulesPanel(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m.aliases = map[string]string{"gc": "get all from corpse"}
	m.runSlash("/highlight red goblin")
	m.runSlash("/trigger hungry = eat bread")

	// Ctrl+A opens it.
	m = drive(m, tea.KeyMsg{Type: tea.KeyCtrlA})
	if !m.rulesOpen {
		t.Fatal("ctrl+a should open the RULES panel")
	}
	if es := m.ruleEntries(); len(es) != 3 {
		t.Fatalf("expected 3 entries (alias, highlight, trigger), got %d", len(es))
	}

	view := ansi.Strip(m.View())
	for _, want := range []string{"RULES", "ALIASES", "gc", "HIGHLIGHTS", "goblin", "TRIGGERS", "hungry"} {
		if !strings.Contains(view, want) {
			t.Errorf("RULES view missing %q", want)
		}
	}

	// Navigate to the trigger (index 2) and edit → loads the command, closes.
	m = drive(m, tea.KeyMsg{Type: tea.KeyDown}, tea.KeyMsg{Type: tea.KeyDown}, tea.KeyMsg{Type: tea.KeyEnter})
	if m.rulesOpen {
		t.Error("editing should close the panel")
	}
	if m.in.Value() != "/trigger hungry = eat bread" {
		t.Errorf("edit loaded %q into the input", m.in.Value())
	}

	// Reopen and delete the alias (cursor 0) with d.
	m = drive(m, tea.KeyMsg{Type: tea.KeyCtrlA})
	m.rulesCursor = 0
	m = drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if _, ok := m.aliases["gc"]; ok {
		t.Error("d should delete the selected alias")
	}
}
