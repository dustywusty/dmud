package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestFuzzyMatch(t *testing.T) {
	cases := []struct {
		q, target string
		want      bool
	}{
		{"", "anything", true},
		{"kr", "kill rat", true},   // subsequence
		{"cast", "cast", true},     // exact
		{"CAST", "cast fire", true}, // case-insensitive
		{"zzz", "cast", false},
		{"tin", "inventory", false}, // order matters
	}
	for _, c := range cases {
		if got := fuzzyMatch(c.q, c.target); got != c.want {
			t.Errorf("fuzzyMatch(%q,%q) = %v, want %v", c.q, c.target, got, c.want)
		}
	}
}

func TestPaletteRefilterRanksPrefix(t *testing.T) {
	p := &cmdPalette{all: []string{"sell", "cast", "score", "scan"}}
	p.query = "sc"
	p.refilter()
	if len(p.filtered) < 2 {
		t.Fatalf("expected matches for 'sc', got %v", p.filtered)
	}
	// "scan"/"score" start with "sc" and should rank before a mere subsequence.
	if !strings.HasPrefix(p.filtered[0], "sc") {
		t.Errorf("prefix match should rank first, got %v", p.filtered)
	}
}

func TestPaletteOpenAndInsert(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 40})

	// Ctrl+K opens the palette.
	mm, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlK})
	m = mm.(model)
	if m.cmdpal == nil {
		t.Fatal("ctrl+k should open the command palette")
	}
	if view := ansi.Strip(m.View()); !strings.Contains(view, "command palette") {
		t.Error("palette overlay should render")
	}

	// Type a query, then Enter inserts the top match into the input.
	for _, r := range "inv" {
		mm, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = mm.(model)
	}
	if len(m.cmdpal.filtered) == 0 {
		t.Fatal("query 'inv' should match at least 'inventory'")
	}
	mm, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(model)
	if m.cmdpal != nil {
		t.Error("Enter should close the palette")
	}
	if !strings.HasPrefix(m.in.Value(), "inv") {
		t.Errorf("Enter should insert the selection into the input, got %q", m.in.Value())
	}
}
