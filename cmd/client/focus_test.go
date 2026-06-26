package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTabCyclesAllPanes(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	tab := tea.KeyMsg{Type: tea.KeyTab}

	// focusables in display order: Here, Main, Map, Comms (default focus is Main).
	for i, want := range []focusTarget{focusMap, focusChat, focusHere, focusMain} {
		m = drive(m, tab)
		if m.focus != want {
			t.Errorf("tab %d → %v, want %v", i+1, m.focus, want)
		}
	}

	// After the loop focus is back on OUTPUT; Shift+Tab steps back to HERE.
	m = drive(m, tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.focus != focusHere {
		t.Errorf("shift+tab → %v, want HERE", m.focus)
	}
}

func TestResizeFocusedPane(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 40})

	// Resize the HERE column: focus it, enter layout mode, grow with Right.
	m.focus = focusHere
	startLeft := m.wantLeft
	m = drive(m, tea.KeyMsg{Type: tea.KeyCtrlO}, tea.KeyMsg{Type: tea.KeyRight})
	if m.wantLeft != startLeft+4 {
		t.Errorf("HERE width: wantLeft %d → %d, want +4", startLeft, m.wantLeft)
	}

	// Switch focus to MAP and grow its height with Up.
	m.focus = focusMap
	startMap := m.wantMapH
	m = drive(m, tea.KeyMsg{Type: tea.KeyUp})
	if m.wantMapH != startMap+2 {
		t.Errorf("MAP height: wantMapH %d → %d, want +2", startMap, m.wantMapH)
	}
}
