package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestEventRoomContents(t *testing.T) {
	raw := `EVENT|{"type":"room.contents","players":["Gandalf"],"npcs":["a sneaky goblin"],"items":["a cookie x3"],"corpses":["the corpse of a rat"]}`

	r := classify(raw)
	if r.contents == nil {
		t.Fatal("EVENT|room.contents should decode into contents")
	}
	if r.toMain != "" || r.toChat != "" || r.status != nil {
		t.Errorf("a structured event must be silent, got %+v", r)
	}

	// Unknown event types are ignored (forward-compatible).
	if r2 := classify(`EVENT|{"type":"char.vitals","hp":5}`); r2.contents != nil || r2.toMain != "" {
		t.Errorf("unknown event types should be ignored, got %+v", r2)
	}
}

func TestEventCharVitalsFxAndGold(t *testing.T) {
	raw := `EVENT|{"type":"char.vitals","hp":40,"max_hp":100,"level":2,"xp":0,"req_xp":100,"area":"X","gold":57,` +
		`"fx":[{"name":"Burning","kind":"dot","remaining":6,"magnitude":-4},{"name":"Blessing","kind":"buff","remaining":0,"magnitude":10}]}`

	r := classify(raw)
	if r.status == nil {
		t.Fatal("char.vitals should decode into a status update")
	}
	if r.status.Gold != 57 {
		t.Errorf("gold = %d, want 57", r.status.Gold)
	}
	if len(r.status.Fx) != 2 {
		t.Fatalf("fx len = %d, want 2", len(r.status.Fx))
	}
	if f := r.status.Fx[0]; f.Name != "Burning" || f.Kind != "dot" || f.Remaining != 6 {
		t.Errorf("fx[0] = %+v, want Burning/dot/6", f)
	}

	// A DoT with a timer renders its name and countdown; legacy names still work.
	if got := fxLabel(r.status.Fx, nil); !strings.Contains(got, "Burning") || !strings.Contains(got, "6s") {
		t.Errorf("fxLabel = %q, want Burning + 6s", got)
	}
	if got := fxLabel(nil, []string{"bless"}); !strings.Contains(got, "bless") {
		t.Errorf("fxLabel legacy fallback = %q, want bless", got)
	}

	// Gold shows on the rendered status bar.
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 140, Height: 40}, chunkMsg{raw: raw})
	if view := ansi.Strip(m.View()); !strings.Contains(view, "57 gold") {
		t.Error("status bar should render the gold amount")
	}
}

func TestEventCharVitals(t *testing.T) {
	raw := `EVENT|{"type":"char.vitals","hp":42,"max_hp":100,"level":3,"xp":120,"req_xp":300,"area":"Whispering Woods","effects":["bless"]}`

	r := classify(raw)
	if r.status == nil {
		t.Fatal("char.vitals should decode into a status update")
	}
	s := r.status
	if s.HP != 42 || s.MaxHP != 100 || s.Level != 3 || s.XP != 120 || s.ReqXP != 300 {
		t.Errorf("vitals parsed wrong: %+v", s)
	}
	if s.Area != "Whispering Woods" || len(s.Effects) != 1 || s.Effects[0] != "bless" {
		t.Errorf("area/effects wrong: %+v", s)
	}

	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 40}, chunkMsg{raw: raw})
	if !m.hasGot || m.status.HP != 42 {
		t.Errorf("status bar not updated from char.vitals: %+v", m.status)
	}
	if view := ansi.Strip(m.View()); !strings.Contains(view, "42/100") {
		t.Error("status bar should render 42/100 from char.vitals")
	}
}

func TestEventEnduranceBar(t *testing.T) {
	// char.vitals carrying endurance shows an EN bar; without it (max_ep 0) the
	// bar is hidden, so older servers don't render an empty gauge.
	withEP := `EVENT|{"type":"char.vitals","hp":42,"max_hp":100,"ep":30,"max_ep":120,"level":1,"xp":0,"req_xp":100,"area":"X"}`
	r := classify(withEP)
	if r.status == nil || !r.status.HasEP || r.status.EP != 30 || r.status.MaxEP != 120 {
		t.Fatalf("endurance not parsed: %+v", r.status)
	}

	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 140, Height: 40}, chunkMsg{raw: withEP})
	if view := ansi.Strip(m.View()); !strings.Contains(view, "EN") || !strings.Contains(view, "30/120") {
		t.Errorf("status bar should show the EN gauge 30/120; view:\n%s", view)
	}

	// No endurance field → no EN gauge.
	noEP := `EVENT|{"type":"char.vitals","hp":42,"max_hp":100,"level":1,"xp":0,"req_xp":100,"area":"X"}`
	m2 := newModel(newConn(transportWS, "x"), nil)
	m2 = drive(m2, tea.WindowSizeMsg{Width: 140, Height: 40}, chunkMsg{raw: noEP})
	if r := classify(noEP); r.status.HasEP {
		t.Error("a vitals frame without max_ep should not flag endurance")
	}
}

func TestEventStatsOnStatusBar(t *testing.T) {
	raw := `EVENT|{"type":"char.vitals","hp":50,"max_hp":50,"level":1,"xp":0,"req_xp":100,"area":"X",` +
		`"stats":{"STR":20,"DEX":4,"CON":20,"INT":2,"WIS":6},"race":"Ogre"}`
	r := classify(raw)
	if r.status == nil || r.status.Stats["STR"] != 20 || r.status.Race != "Ogre" {
		t.Fatalf("stats not parsed: %+v", r.status)
	}

	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 170, Height: 40}, chunkMsg{raw: raw})
	if view := ansi.Strip(m.View()); !strings.Contains(view, "STR20") || !strings.Contains(view, "INT2") {
		t.Errorf("status bar should show the stat segment (STR20 … INT2); view:\n%s", view)
	}
}

func TestEventMacrosLoadBar(t *testing.T) {
	raw := `EVENT|{"type":"macros","source":"class:cleric","set":[` +
		`{"slot":1,"label":"mend","cmd":"cast mend"},` +
		`{"slot":2,"label":"smite","cmd":"cast smite"}]}`
	r := classify(raw)
	if len(r.macros) != 2 || r.macros[0].Cmd != "cast mend" {
		t.Fatalf("macros not parsed: %+v", r.macros)
	}

	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 40}, chunkMsg{raw: raw})

	if m.macros["f1"].Cmd != "cast mend" || m.macros["f2"].Label != "smite" {
		t.Errorf("class macros not applied to the bar: %+v", m.macros)
	}
	if m.macroH != 1 {
		t.Error("the macro bar should appear once a loadout is pushed")
	}
	if view := ansi.Strip(m.View()); !strings.Contains(view, "mend") || !strings.Contains(view, "smite") {
		t.Errorf("the bar should show the pushed macros; view:\n%s", view)
	}
}

func TestEventUpdatesHereLive(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m,
		tea.WindowSizeMsg{Width: 120, Height: 40},
		// No look/room block — just the pushed event.
		chunkMsg{raw: `EVENT|{"type":"room.contents","players":["Gandalf"],"npcs":["a sneaky goblin"],"items":[],"corpses":[]}`},
	)

	if got := strings.Join(m.room.Occupants, "|"); got != "Gandalf|a sneaky goblin" {
		t.Errorf("HERE = %q, want \"Gandalf|a sneaky goblin\"", got)
	}
	if !m.roster["Gandalf"] {
		t.Error("a present player should be marked in the roster")
	}

	view := ansi.Strip(m.View())
	if !strings.Contains(view, "Gandalf") || !strings.Contains(view, "sneaky goblin") {
		t.Errorf("HERE panel should render the pushed occupants; view:\n%s", view)
	}

	// A later event replaces the list (someone left, a goblin spawned).
	m = drive(m, chunkMsg{raw: `EVENT|{"type":"room.contents","players":[],"npcs":["a sneaky goblin","a town guard"],"items":[],"corpses":[]}`})
	if got := strings.Join(m.room.Occupants, "|"); got != "a sneaky goblin|a town guard" {
		t.Errorf("HERE after update = %q", got)
	}
}

func TestEventRoomInfo(t *testing.T) {
	raw := `EVENT|{"type":"room.info","name":"Whispering Woods","exits":["north","east"],"region":"Forest"}`

	r := classify(raw)
	if r.info == nil {
		t.Fatal("room.info should decode into info")
	}
	if r.info.Title != "Whispering Woods" || strings.Join(r.info.Exits, ",") != "north,east" {
		t.Errorf("room.info parsed wrong: %+v", r.info)
	}
	if r.toMain != "" || r.status != nil || r.room != nil {
		t.Errorf("room.info must be silent, got %+v", r)
	}

	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 120, Height: 40}, chunkMsg{raw: raw})
	if m.room.Title != "Whispering Woods" {
		t.Errorf("room title not set from room.info: %q", m.room.Title)
	}
	if got := strings.Join(m.room.Exits, ","); got != "north,east" {
		t.Errorf("exits not set from room.info: %q", got)
	}
	if m.mapper.cur != "Whispering Woods" {
		t.Errorf("mapper should arrive from room.info: cur=%q", m.mapper.cur)
	}
	if view := ansi.Strip(m.View()); !strings.Contains(view, "Whispering Woods") {
		t.Error("MAP header should show the room name from room.info")
	}
}

func TestRenderComms(t *testing.T) {
	cases := []struct {
		ev       commsMsg
		wantTab  string
		wantLine string
	}{
		{commsMsg{Channel: "say", Self: true, Text: "hi"}, "Local", "You say: hi"},
		{commsMsg{Channel: "say", From: "Gandalf", Text: "hi"}, "Local", "Gandalf says: hi"},
		{commsMsg{Channel: "shout", Self: true, Text: "oi"}, "Shout", "You shout: oi"},
		{commsMsg{Channel: "shout", From: "Gandalf", Text: "oi"}, "Shout", "Gandalf shouts: oi"},
		{commsMsg{Channel: "tell", Self: true, To: "Bob", Text: "psst"}, "Bob", "You tell Bob: psst"},
		{commsMsg{Channel: "tell", From: "Bob", Text: "psst"}, "Bob", "Bob tells you: psst"},
	}
	for _, c := range cases {
		tab, line := renderComms(&c.ev)
		if tab != c.wantTab || line != c.wantLine {
			t.Errorf("renderComms(%+v) = (%q,%q), want (%q,%q)", c.ev, tab, line, c.wantTab, c.wantLine)
		}
	}
}

func TestEventCommsRouting(t *testing.T) {
	// A structured comms event routes only to r.comms — no plain-text duplicate.
	r := classify(`EVENT|{"type":"comms","channel":"say","from":"X","text":"y"}`)
	if r.comms == nil || r.toMain != "" || r.toChat != "" {
		t.Errorf("comms event should route only to r.comms, got %+v", r)
	}

	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m,
		tea.WindowSizeMsg{Width: 120, Height: 40},
		chunkMsg{raw: `EVENT|{"type":"comms","channel":"tell","from":"Bob","text":"psst"}`},
	)
	tab := m.comms.find("Bob")
	if !strings.Contains(ansi.Strip(tab.raw), "Bob tells you: psst") {
		t.Errorf("a tell should land in the sender's tab; raw=%q", ansi.Strip(tab.raw))
	}
}

func TestEventCombat(t *testing.T) {
	raw := `EVENT|{"type":"combat","target":"a goblin","damage":7,"target_hp":12,"target_max":20,"killed":false}`

	r := classify(raw)
	if r.combat == nil {
		t.Fatal("combat should decode into combat")
	}
	if r.combat.Target != "a goblin" || r.combat.TargetHP != 12 || r.combat.TargetMax != 20 {
		t.Errorf("combat parsed wrong: %+v", r.combat)
	}
	if r.toMain != "" || r.status != nil {
		t.Errorf("combat event must be silent, got %+v", r)
	}

	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 140, Height: 40},
		chunkMsg{raw: `EVENT|{"type":"char.vitals","hp":50,"max_hp":50,"level":1,"xp":0,"req_xp":100,"area":"X"}`},
		chunkMsg{raw: raw})
	if m.target.Target != "a goblin" || m.target.TargetHP != 12 {
		t.Errorf("combat target not stored: %+v", m.target)
	}
	if view := ansi.Strip(m.View()); !strings.Contains(view, "a goblin") || !strings.Contains(view, "12/20") {
		t.Errorf("status bar should show the enemy HP bar; view:\n%s", view)
	}

	// A kill marks the target slain.
	m = drive(m, chunkMsg{raw: `EVENT|{"type":"combat","target":"a goblin","damage":12,"target_hp":0,"target_max":20,"killed":true}`})
	if seg := ansi.Strip(targetSeg(m.target)); !strings.Contains(seg, "slain") {
		t.Errorf("a killed target should render as slain, got %q", seg)
	}

	// Moving to a new room clears the enemy bar.
	m = drive(m, chunkMsg{raw: `EVENT|{"type":"room.info","name":"Elsewhere","exits":["south"]}`})
	if m.target.Target != "" {
		t.Errorf("entering a new room should clear the target, got %+v", m.target)
	}
}
