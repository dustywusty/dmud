package main

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCommsTabAt(t *testing.T) {
	c := newComms() // tabs: "Local" (5), " · " (3), "Shout" (5)
	cases := map[int]int{
		0:  0,  // Local
		4:  0,  // Local
		6:  -1, // separator
		8:  1,  // Shout
		12: 1,  // Shout
		99: -1, // past the end
	}
	for offset, want := range cases {
		if got := c.tabAt(offset); got != want {
			t.Errorf("tabAt(%d) = %d, want %d", offset, got, want)
		}
	}
}

func TestViewportAtRegions(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	// Geometry: middleTop=4, midH=33, colLeft=26, colMain=50 (main x 26..75),
	// right column x>=76, mapH=13 so COMMS starts at y=17.

	if m.viewportAt(50, 20) != &m.main {
		t.Error("center column should map to the main viewport")
	}
	if m.viewportAt(90, 25) != &m.comms.vp {
		t.Error("right column below the map should map to COMMS")
	}
	if m.viewportAt(90, 10) != nil {
		t.Error("the MAP box is not scrollable")
	}
	if m.viewportAt(10, 20) != &m.here {
		t.Error("the left column should map to the HERE viewport")
	}
	if m.viewportAt(50, 1) != nil {
		t.Error("header/status rows are not scrollable")
	}
}

func TestClickFocusAndTabSwitch(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 40})

	m.focus = focusChat
	m.handleClick(50, 20) // center column
	if m.focus != focusMain {
		t.Error("clicking the main panel should focus it")
	}

	m.handleClick(90, 25) // COMMS body
	if m.focus != focusChat {
		t.Error("clicking COMMS should focus it")
	}

	// Tab bar is at y = middleTop+mapH+1 = 18; COMMS inner starts at x=77.
	// "Shout" sits at offset 8.. so x≈86.
	m.handleClick(86, 18)
	if m.comms.activeTab().name != "Shout" {
		t.Errorf("clicking a tab should switch to it; active=%q", m.comms.activeTab().name)
	}
}
