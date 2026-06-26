package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestApplyThemeFallback(t *testing.T) {
	defer applyTheme("default") // don't leak palette into other tests

	if len(themeNames()) < 2 {
		t.Fatal("expected several named themes")
	}
	for _, name := range themeNames() {
		if _, ok := palettes[name]; !ok {
			t.Errorf("themeNames lists %q but it has no palette", name)
		}
	}
	// An unknown theme must fall back to default without panicking.
	applyTheme("does-not-exist")
}

func TestRenderPromptLine(t *testing.T) {
	s := statusInfo{HP: 40, MaxHP: 100, EP: 10, MaxEP: 50, XP: 5, ReqXP: 100, Level: 3, Gold: 12, Area: "Cave"}
	got := renderPromptLine("HP {hp}/{maxhp} EN {ep} L{lvl} {gold}g @{area}", s)
	want := " HP 40/100 EN 10 L3 12g @Cave"
	if got != want {
		t.Errorf("renderPromptLine = %q, want %q", got, want)
	}
}

func TestCustomPromptRendersOnBar(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m = drive(m, vitals(40, 100)) // populates status + hasGot
	m.prompt = "MYHP {hp}/{maxhp}"
	if view := ansi.Strip(m.View()); !strings.Contains(view, "MYHP 40/100") {
		t.Error("a custom prompt should render on the status bar")
	}
}
