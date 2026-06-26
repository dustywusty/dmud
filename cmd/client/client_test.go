package main

import (
	"os"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// drive applies a sequence of messages to the model and returns the result.
func drive(m model, msgs ...tea.Msg) model {
	var tm tea.Model = m
	for _, msg := range msgs {
		tm, _ = tm.Update(msg)
	}
	return tm.(model)
}

func TestClassify(t *testing.T) {
	t.Run("state frame", func(t *testing.T) {
		r := classify("STATE|HP:42/100|LEVEL:3|XP:120/300|AREA:Whispering Woods|EFFECTS:bless:5,haste:0")
		if r.status == nil {
			t.Fatal("expected a status update")
		}
		if r.status.HP != 42 || r.status.MaxHP != 100 {
			t.Errorf("HP = %d/%d, want 42/100", r.status.HP, r.status.MaxHP)
		}
		if r.status.Level != 3 || r.status.XP != 120 || r.status.ReqXP != 300 {
			t.Errorf("level/xp parsed wrong: %+v", r.status)
		}
		if r.status.Area != "Whispering Woods" {
			t.Errorf("area = %q", r.status.Area)
		}
		if len(r.status.Effects) != 2 || r.status.Effects[0] != "bless" {
			t.Errorf("effects = %v", r.status.Effects)
		}
		if r.toMain != "" {
			t.Errorf("STATE should not echo to main, got %q", r.toMain)
		}
	})

	t.Run("state area with embedded newlines is collapsed to title", func(t *testing.T) {
		// The live server truncates the full room description into AREA.
		r := classify("STATE|HP:100/100|LEVEL:1|XP:0/100|AREA:THE BLIGHTED CROSSROADS\n\nYou stand at a misty cro...")
		if r.status == nil {
			t.Fatal("expected a status update")
		}
		if r.status.Area != "THE BLIGHTED CROSSROADS" {
			t.Errorf("area = %q, want clean first line", r.status.Area)
		}
		if strings.ContainsAny(r.status.Area, "\r\n") {
			t.Error("area still contains a newline; status bar would break")
		}
	})

	t.Run("room block", func(t *testing.T) {
		chunk := "THE BLIGHTED CROSSROADS\n\nYou stand at a misty crossroads.\n\nGandalf is here.\na cookie x3 is here.\n\nExits: [north, east, south, west]"
		r := classify(chunk)
		if r.room == nil {
			t.Fatal("expected a room update")
		}
		if r.room.Title != "THE BLIGHTED CROSSROADS" {
			t.Errorf("title = %q", r.room.Title)
		}
		if got := strings.Join(r.room.Exits, ","); got != "north,east,south,west" {
			t.Errorf("exits = %q", got)
		}
		if got := strings.Join(r.room.Occupants, "|"); got != "Gandalf|a cookie" {
			t.Errorf("occupants = %q, want \"Gandalf|a cookie\" (count stripped)", got)
		}
		if r.toMain == "" {
			t.Error("room body should still echo to main")
		}
	})

	t.Run("chat line", func(t *testing.T) {
		for _, line := range []string{
			"Gandalf says: hello there",
			"You say: hi",
			"Frodo shouts: help!",
		} {
			if r := classify(line); r.toChat == "" {
				t.Errorf("%q should route to chat", line)
			}
		}
	})

	t.Run("plain output to main", func(t *testing.T) {
		r := classify("A goblin snarls at you.")
		if r.toMain == "" || r.toChat != "" || r.status != nil {
			t.Errorf("plain line misrouted: %+v", r)
		}
	})

	t.Run("STATUS notification strips its prefix", func(t *testing.T) {
		r := classify("STATUS|You gained 75 experience!")
		got := ansi.Strip(r.toMain)
		if !strings.Contains(got, "You gained 75 experience!") {
			t.Errorf("notice text missing: %q", got)
		}
		if strings.Contains(got, "STATUS|") {
			t.Errorf("STATUS| prefix leaked into output: %q", got)
		}
	})

	t.Run("DMG combat strips tag and DEATH status", func(t *testing.T) {
		r := classify("DMG|You attacked a sneaky goblin for 49 damage!")
		if got := ansi.Strip(r.toMain); !strings.Contains(got, "You attacked") || strings.Contains(got, "DMG") {
			t.Errorf("combat line misrouted: %q", got)
		}
		r2 := classify("DMG|DEATH|a sneaky goblin has been slain!")
		got2 := ansi.Strip(r2.toMain)
		if !strings.Contains(got2, "has been slain") || strings.Contains(got2, "DEATH") || strings.Contains(got2, "DMG") {
			t.Errorf("death line misrouted: %q", got2)
		}
	})
}

func TestViewRendersPanels(t *testing.T) {
	m := newModel(newConn(transportWS, "localhost:8080"), nil)
	m = drive(m,
		tea.WindowSizeMsg{Width: 120, Height: 40},
		connectedMsg{},
		chunkMsg{raw: "STATUS|Gandalf has joined the game."}, // marks Gandalf as a player
		chunkMsg{raw: "THE BLIGHTED CROSSROADS\n\nA misty crossroads.\n\nGandalf is here.\na sleepy cat is here.\n\nExits: [north, east]"},
		chunkMsg{raw: "STATE|HP:42/100|LEVEL:3|XP:120/300|AREA:Whispering Woods"},
		chunkMsg{raw: "Gandalf says: well met"},
		chunkMsg{raw: "A goblin appears."},
	)

	view := ansi.Strip(m.View())

	for _, want := range []string{
		"THE BLIGHTED CROSSROADS", // map header + main
		"OUTPUT",                  // main panel title
		"Local",                   // COMMS tab bar (say goes to Local)
		"Shout",                   // COMMS tab bar
		"exits:",                  // map header exits line
		"[@]",                     // current-room box on the map
		"HERE",                    // occupants panel title
		"sleepy cat",              // occupant (NPC/item)
		"◆",                       // player highlight marker for Gandalf
		"HP", "42/100",            // status bar
		"Lv 3",
		"Gandalf",   // chat content
		"A goblin",  // main content
		"connected", // header
	} {
		if !strings.Contains(view, want) {
			t.Errorf("rendered view is missing %q", want)
		}
	}
}

func TestConfigRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	want := clientConfig{
		Layout:     layoutPrefs{LeftWidth: 30, RightWidth: 60, MapHeight: 8, ShowRight: false},
		Compact:    true,
		Aliases:    map[string]string{"gc": "get all from corpse"},
		Highlights: []ruleConfig{{Pattern: "tells you", Value: "213"}},
		Triggers:   []ruleConfig{{Pattern: "You are hungry", Value: "eat bread"}},
	}
	if err := saveConfig(want); err != nil {
		t.Fatalf("saveConfig: %v", err)
	}
	if got := loadConfig(); !reflect.DeepEqual(got, want) {
		t.Errorf("round-trip = %+v, want %+v", got, want)
	}

	// A missing config returns defaults rather than zero values.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if got := loadConfig(); !reflect.DeepEqual(got, defaultConfig()) {
		t.Errorf("missing file = %+v, want defaults %+v", got, defaultConfig())
	}
}

func TestRosterTracking(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)

	m.updateRoster("STATUS|Gandalf has joined the game.")
	if !m.roster["Gandalf"] {
		t.Error("join notice should add the player to the roster")
	}
	m.updateRoster("STATUS|Gandalf has left the game.")
	if m.roster["Gandalf"] {
		t.Error("leave notice should remove the player from the roster")
	}

	who := strings.Join([]string{
		"┌────────┬──────┬───────┬──────────────┐",
		"│ Player │ Race │ Level │ Online Since │",
		"├────────┼──────┼───────┼──────────────┤",
		"│ Frodo  │ ??   │ 3     │ 2 minutes    │",
		"└────────┴──────┴───────┴──────────────┘",
	}, "\n")
	m.updateRoster(who)
	if !m.roster["Frodo"] {
		t.Errorf("who table should enrich the roster; got %v", m.roster)
	}
	if m.roster["Player"] {
		t.Error("the table header should not be treated as a player")
	}
}

// TestDumpLayout prints the rendered UI (ANSI stripped) for visual inspection.
// Run with: DMUD_DUMP=1 go test ./cmd/client/ -run TestDumpLayout -v
func TestDumpLayout(t *testing.T) {
	if os.Getenv("DMUD_DUMP") != "1" {
		t.Skip("set DMUD_DUMP=1 to print a layout snapshot")
	}
	m := newModel(newConn(transportWS, "localhost:8080"), nil)
	m = drive(m,
		tea.WindowSizeMsg{Width: 116, Height: 34},
		connectedMsg{},
		chunkMsg{raw: "STATUS|Gandalf has joined the game."},
		chunkMsg{raw: "THE BLIGHTED CROSSROADS\n\nYou stand at a misty crossroads where ancient paths converge. Twisted trees frame the scene.\n\nExits: [north, east, south, west]"},
	)
	m.mapper.noteCommand("north")
	m = drive(m, chunkMsg{raw: "TWISTED FOREST EDGE\n\nThe path leads to the threshold of a dense forest.\n\na sneaky goblin is here.\n\nExits: [north, south, east, down]"})
	m.mapper.noteCommand("east")
	m = drive(m,
		chunkMsg{raw: "MUSHROOM GROVE\n\nGlowing fungi light a damp hollow.\n\nGandalf is here.\nthe corpse of a goblin is here.\na cookie x3 is here.\n\nExits: [west, north, up]"},
		chunkMsg{raw: "STATE|HP:566/600|LEVEL:1|XP:75/100|AREA:MUSHROOM GROVE|EFFECTS:bless:5"},
		chunkMsg{raw: "Gandalf says: careful, the goblins bite"},
		chunkMsg{raw: "You say: noted, thanks"},
		chunkMsg{raw: "Frodo shouts: anyone near the bakehouse?"},
		chunkMsg{raw: "Wylie tells you: bring cookies, love"},
		chunkMsg{raw: "DMG|a sneaky goblin attacked you for 13 damage!"},
		chunkMsg{raw: "DMG|You attacked a sneaky goblin for 49 damage!"},
		chunkMsg{raw: "STATUS|You gained 75 experience!"},
	)
	t.Logf("normal layout:\n%s", ansi.Strip(m.View()))

	m.runSlash("/map")
	t.Logf("full-screen map (/map):\n%s", ansi.Strip(m.View()))

	m.fullMap = false
	m.aliases = map[string]string{"gc": "get all from corpse", "k": "kill"}
	m.runSlash("/highlight yellow tells you")
	m.runSlash("/trigger low health = quaff red")
	m.rulesOpen = true
	t.Logf("RULES panel (^A):\n%s", ansi.Strip(m.View()))
}

func TestEnterSendsAndEchoes(t *testing.T) {
	m := newModel(newConn(transportWS, "localhost:8080"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 30})

	// type "look" then Enter
	for _, r := range "look" {
		m = drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m = drive(m, tea.KeyMsg{Type: tea.KeyEnter})

	if len(m.history) != 1 || m.history[0] != "look" {
		t.Errorf("history = %v, want [look]", m.history)
	}
	if m.in.Value() != "" {
		t.Errorf("input not cleared: %q", m.in.Value())
	}
	if !strings.Contains(ansi.Strip(m.View()), "look") {
		t.Error("command echo missing from output")
	}
}
