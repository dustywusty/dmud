package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestEquipmentEventAndGearOverlay(t *testing.T) {
	raw := `EVENT|{"type":"equipment","slots":[` +
		`{"slot":"weapon","name":"Rusty Dagger","damage":"2-5"},` +
		`{"slot":"armor","name":"Leather Chestpiece","armor":3,"hp_bonus":15}]}`

	// The event is silent (doesn't echo to OUTPUT) but updates equipment state.
	r := classify(raw)
	if !r.hasEquip || len(r.equip) != 2 {
		t.Fatalf("equipment event should decode 2 slots, got hasEquip=%v len=%d", r.hasEquip, len(r.equip))
	}
	if r.toMain != "" {
		t.Errorf("equipment event must be silent, got toMain=%q", r.toMain)
	}

	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 40}, chunkMsg{raw: raw})
	if len(m.equipment) != 2 {
		t.Fatalf("model should hold 2 equipped items, got %d", len(m.equipment))
	}

	// Ctrl+E opens the gear overlay and renders the worn items.
	mm, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlE})
	m = mm.(model)
	if !m.gearOpen {
		t.Fatal("ctrl+e should open the gear overlay")
	}
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "Rusty Dagger") || !strings.Contains(view, "Leather Chestpiece") {
		t.Errorf("gear overlay should list worn items, got:\n%s", view)
	}
	if !strings.Contains(view, "2-5") || !strings.Contains(view, "armor 3") {
		t.Error("gear overlay should show item modifiers")
	}

	// Esc closes it.
	mm, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = mm.(model)
	if m.gearOpen {
		t.Error("esc should close the gear overlay")
	}
}
