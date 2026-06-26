package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestExpandSpeedwalk(t *testing.T) {
	cases := map[string][]string{
		"3n2e": {"n", "n", "n", "e", "e"},
		"ne":   {"n", "e"},
		"n":    {"n"},    // single direction stays a plain move
		"look": {"look"}, // ordinary command untouched
		"2u":   {"u", "u"},
	}
	for in, want := range cases {
		if got := expandSpeedwalk(in); strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("expandSpeedwalk(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestAliasExpansion(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m.aliases = map[string]string{
		"gc": "get all from corpse",
		"ka": "kill goblin;cast heal",
		"g":  "get",
	}

	if got := m.expand("gc"); len(got) != 1 || got[0] != "get all from corpse" {
		t.Errorf("alias gc → %v", got)
	}
	if got := m.expand("ka"); len(got) != 2 || got[0] != "kill goblin" || got[1] != "cast heal" {
		t.Errorf("alias ka → %v", got)
	}
	if got := m.expand("g cookie"); len(got) != 1 || got[0] != "get cookie" {
		t.Errorf("alias g + args → %v", got)
	}
}

func TestTabCompletion(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)

	m.in.SetValue("wh")
	m.completeInput()
	if m.in.Value() != "who" {
		t.Errorf("complete \"wh\" → %q, want \"who\"", m.in.Value())
	}

	m.roster["Gandalf"] = true
	m.in.SetValue("tell Gand")
	m.completeInput()
	if m.in.Value() != "tell Gandalf" {
		t.Errorf("roster completion → %q, want \"tell Gandalf\"", m.in.Value())
	}
}

func TestSlashAliasHighlightTrigger(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)

	m.runSlash("/alias gc get all from corpse")
	if m.aliases["gc"] != "get all from corpse" {
		t.Errorf("/alias not set: %v", m.aliases)
	}

	m.runSlash("/highlight red goblin")
	if len(m.highlights) != 1 {
		t.Fatalf("/highlight not added: %v", m.highlights)
	}
	// lipgloss emits no ANSI in a non-TTY test, so assert on the compiled rule.
	if h := m.highlights[0]; h.Value != "203" || !h.re.MatchString("a goblin appears") {
		t.Errorf("highlight rule wrong: %+v", h)
	}

	m.runSlash("/trigger hungry = eat bread")
	if got := matchTriggers("you are hungry now", m.triggers); len(got) != 1 || got[0] != "eat bread" {
		t.Errorf("trigger did not fire: %v", got)
	}
}

func TestResizeModeUsesPlainArrows(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	start := m.wantRight

	// Ctrl+O enters layout mode; plain Right/Up resize; Esc exits.
	m = drive(m, tea.KeyMsg{Type: tea.KeyCtrlO})
	if !m.layoutMode {
		t.Fatal("ctrl+o should enter layout mode")
	}
	m = drive(m, tea.KeyMsg{Type: tea.KeyRight})
	if m.wantRight != start+4 {
		t.Errorf("right arrow should widen: wantRight %d → %d", start, m.wantRight)
	}
	m = drive(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.layoutMode {
		t.Error("esc should leave layout mode")
	}
}

func TestAltWordKeysMoveEastWest(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	// macOS sends Option+Left/Right as alt+b / alt+f.
	m = drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}, Alt: true})
	m = drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}, Alt: true})
	if got := strings.Join(m.mapper.pending, ","); got != "west,east" {
		t.Errorf("alt+b/alt+f should queue west,east moves; got %q", got)
	}
}

func TestSlashMapToggleAndClear(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)

	m.runSlash("/map")
	if !m.fullMap {
		t.Error("/map should enable the full-screen map")
	}
	m.runSlash("/map")
	if m.fullMap {
		t.Error("/map should toggle the full-screen map off")
	}

	m.appendMain("hello world")
	m.runSlash("/clear")
	if m.mainRaw != "" {
		t.Errorf("/clear left content: %q", m.mainRaw)
	}
}

func TestSlashMapReset(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := newModel(newConn(transportWS, "x"), nil)
	m.mapper.arrive("Old Room", []string{"east"})
	if len(m.mapper.rooms) == 0 {
		t.Fatal("precondition: map should have a room")
	}
	m.room = roomInfo{Title: "Current Room", Exits: []string{"north"}}

	m.runSlash("/map reset")

	if _, ok := m.mapper.rooms["Old Room"]; ok {
		t.Error("/map reset should drop the old map")
	}
	if _, ok := m.mapper.rooms["Current Room"]; !ok {
		t.Error("/map reset should re-seed the current room")
	}
}
