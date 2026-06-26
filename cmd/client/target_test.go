package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestStripArticle(t *testing.T) {
	cases := map[string]string{
		"a sneaky goblin": "sneaky goblin",
		"an ogre":         "ogre",
		"the rat king":    "rat king",
		"goblin":          "goblin",
	}
	for in, want := range cases {
		if got := stripArticle(in); got != want {
			t.Errorf("stripArticle(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTabTargetCycles(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 40})

	// No enemies present: nothing to attack.
	if cmd := m.handleTarget(""); cmd != nil {
		t.Error("handleTarget with no targets should not dispatch")
	}

	// Populate the room from a contents event.
	raw := `EVENT|{"type":"room.contents","players":[],"npcs":["a sneaky goblin","a giant rat"],"items":[],"corpses":[]}`
	m = drive(m, chunkMsg{raw: raw})
	if len(m.npcs) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(m.npcs))
	}

	// First press targets the first enemy and advances the cursor.
	if cmd := m.handleTarget(""); cmd == nil {
		t.Error("handleTarget should dispatch a kill when targets exist")
	}
	if m.targetIdx != 1 {
		t.Errorf("target index should advance to 1, got %d", m.targetIdx)
	}
	if view := ansi.Strip(m.View()); !strings.Contains(view, "a sneaky goblin") {
		t.Error("output should announce the current target")
	}
}
