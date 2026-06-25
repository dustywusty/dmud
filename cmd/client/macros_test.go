package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestMacroSlotParsing(t *testing.T) {
	// macroSlot is lenient (parses /macro args): accepts digits or f-prefixed.
	for in, want := range map[string]string{
		"1": "f1", "f1": "f1", "F12": "f12", "12": "f12",
		"0": "", "13": "", "f0": "", "x": "", "": "",
	} {
		if got := macroSlot(in); got != want {
			t.Errorf("macroSlot(%q) = %q, want %q", in, got, want)
		}
	}
	// fKeySlot is strict (keypresses): only real F-keys, never bare digits.
	for in, want := range map[string]string{
		"f1": "f1", "f12": "f12", "1": "", "12": "", "f0": "", "f13": "", "a": "",
	} {
		if got := fKeySlot(in); got != want {
			t.Errorf("fKeySlot(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMacroSetClearLabel(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 40})

	m.handleMacro("1 look")
	if got := m.macros["f1"]; got.Cmd != "look" || got.Label != "" {
		t.Errorf("set bare: got %+v, want {Cmd:look}", got)
	}
	// The confirmation uses the Shift label (⇧1), never the internal slot ("F1").
	if msg := ansi.Strip(m.mainRaw); !strings.Contains(msg, "macro set: ⇧1") || strings.Contains(msg, "F1") {
		t.Errorf("confirmation should read '⇧1', not 'F1'; got %q", msg)
	}

	m.handleMacro("2 Heal = cast heal on self")
	if got := m.macros["f2"]; got.Label != "Heal" || got.Cmd != "cast heal on self" {
		t.Errorf("set with label: got %+v", got)
	}

	m.handleMacro("1") // clear
	if _, ok := m.macros["f1"]; ok {
		t.Error("F1 should have been cleared")
	}

	// A bad slot is rejected, leaving macros untouched.
	before := len(m.macros)
	m.handleMacro("99 nope")
	if len(m.macros) != before {
		t.Error("an out-of-range slot should not create a macro")
	}
}

func TestMacroConfigRoundTrip(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 40})
	m.handleMacro("3 Smite = kill rat")

	cfg := m.config()
	if cfg.Macros["f3"].Cmd != "kill rat" || cfg.Macros["f3"].Label != "Smite" {
		t.Fatalf("config did not capture the macro: %+v", cfg.Macros)
	}

	// A fresh model restores it from that config.
	m2 := newModel(newConn(transportWS, "x"), nil)
	m2.applyConfig(cfg)
	if m2.macros["f3"].Cmd != "kill rat" {
		t.Errorf("applyConfig did not restore the macro: %+v", m2.macros)
	}
}

func TestMacroHotkeyFires(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 40})

	// Set the macro by typing the slash command, then press F1.
	for _, r := range "/macro 1 look" {
		m = drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m = drive(m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.macroH != 1 {
		t.Errorf("macro bar should reserve a row once a macro exists; macroH=%d", m.macroH)
	}
	if view := ansi.Strip(m.View()); !strings.Contains(view, "⇧1 look") {
		t.Errorf("macro bar should show '⇧1 look'; view:\n%s", view)
	}

	// F-keys remain a mid-typing fallback for the Shift+digit bindings.
	before := m.mainRaw
	m = drive(m, tea.KeyMsg{Type: tea.KeyF1})
	if added := strings.TrimPrefix(m.mainRaw, before); !strings.Contains(ansi.Strip(added), "» look") {
		t.Errorf("pressing F1 should echo '» look' to OUTPUT; added=%q", ansi.Strip(added))
	}

	// An unbound F-key is a silent no-op (no echo, no panic).
	before = m.mainRaw
	m = drive(m, tea.KeyMsg{Type: tea.KeyF5})
	if m.mainRaw != before {
		t.Error("an unbound F-key should do nothing")
	}
}

func TestMacroShiftKeyFires(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 40})
	m.handleMacro("1 look") // slot f1 — fired by Shift+1, which the terminal sends as "!"

	// Shift+1 ("!") on an empty input fires the macro and is consumed.
	before := m.mainRaw
	m = drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'!'}})
	if added := strings.TrimPrefix(m.mainRaw, before); !strings.Contains(ansi.Strip(added), "» look") {
		t.Errorf("Shift+1 should fire the macro; added=%q", ansi.Strip(added))
	}
	if m.in.Value() != "" {
		t.Errorf("firing a macro should not type the symbol; input=%q", m.in.Value())
	}

	// With text already in the input, "!" is a literal character, not a macro.
	m.in.SetValue("hi")
	before = m.mainRaw
	m = drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'!'}})
	if m.in.Value() != "hi!" {
		t.Errorf("with input present, '!' should type; got %q", m.in.Value())
	}
	if m.mainRaw != before {
		t.Error("'!' while typing must not fire a macro")
	}

	// An unbound shifted symbol types normally even on an empty line.
	m.in.SetValue("")
	m = drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'@'}}) // slot f2, unbound
	if m.in.Value() != "@" {
		t.Errorf("unbound Shift+2 ('@') should type; got %q", m.in.Value())
	}
}

func TestMacroDigitDoesNotFire(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 40})
	m.macros["f1"] = macro{Cmd: "look"}

	// Typing "1" must land in the input, not fire the F1 macro.
	m = drive(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	if m.in.Value() != "1" {
		t.Errorf("digit key should type into the input, got %q", m.in.Value())
	}
	if strings.Contains(m.mainRaw, "» look") {
		t.Error("typing '1' must not fire the F1 macro")
	}
}

func TestMacroInRulesOverlay(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 40})
	m.handleMacro("1 look")

	var entry *ruleEntry
	for i, e := range m.ruleEntries() {
		if e.kind == "macro" {
			entry = &m.ruleEntries()[i]
			break
		}
	}
	if entry == nil {
		t.Fatal("a macro should appear in the RULES overlay")
	}
	if entry.edit != "/macro edit f1" {
		t.Errorf("macro edit line = %q, want \"/macro edit f1\"", entry.edit)
	}

	// Deleting the selected macro removes it and collapses the bar.
	m.rulesCursor = 0
	m.deleteSelectedRule()
	if _, ok := m.macros["f1"]; ok {
		t.Error("deleteSelectedRule should remove the macro")
	}
	if m.macroH != 0 {
		t.Errorf("macro bar should collapse after the last macro is deleted; macroH=%d", m.macroH)
	}
}

func TestMacroEditorMultiline(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 40})

	m.openMacroEditor("f1")
	if m.macroEd == nil || m.macroEd.slot != "f1" {
		t.Fatal("openMacroEditor should open the editor for f1")
	}
	m.macroEd.label.SetValue("Pull")
	m.macroEd.body.SetValue("look\nkill rat\nsay incoming")
	m.saveMacroEditor()

	if m.macroEd != nil {
		t.Error("saving should close the editor")
	}
	got := m.macros["f1"]
	if got.Label != "Pull" || got.Cmd != "look\nkill rat\nsay incoming" {
		t.Fatalf("multi-line macro stored wrong: %+v", got)
	}

	// Firing runs every line in order, each echoed to OUTPUT.
	before := m.mainRaw
	m = drive(m, tea.KeyMsg{Type: tea.KeyF1})
	added := ansi.Strip(strings.TrimPrefix(m.mainRaw, before))
	for _, want := range []string{"» look", "» kill rat", "» say incoming"} {
		if !strings.Contains(added, want) {
			t.Errorf("firing a multi-line macro should echo %q; got:\n%s", want, added)
		}
	}

	// The bar shows the label, not the whole multi-line body.
	bar := ansi.Strip(m.renderMacroBar())
	if !strings.Contains(bar, "⇧1 Pull") || strings.Contains(bar, "kill rat") {
		t.Errorf("bar should show the label only; got %q", bar)
	}
}

func TestMacroEditorOpenAndCancel(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 40})

	// /macro edit <slot> opens the editor.
	m.handleMacro("edit 2")
	if m.macroEd == nil || m.macroEd.slot != "f2" {
		t.Fatalf("/macro edit 2 should open the editor for f2; macroEd=%v", m.macroEd)
	}

	// Esc cancels without saving.
	m.macroEd.body.SetValue("look")
	m = drive(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.macroEd != nil {
		t.Error("Esc should close the editor")
	}
	if _, ok := m.macros["f2"]; ok {
		t.Error("cancelling must not save the macro")
	}
}

func TestMacroEditorClearsOnEmptyBody(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 40})
	m.handleMacro("1 look") // bind first

	m.openMacroEditor("f1")
	m.macroEd.body.SetValue("   \n  ") // whitespace only
	m.saveMacroEditor()
	if _, ok := m.macros["f1"]; ok {
		t.Error("saving an empty body should clear the slot")
	}
}

func TestMacroBarHiddenWhenEmpty(t *testing.T) {
	m := newModel(newConn(transportWS, "x"), nil)
	m = drive(m, tea.WindowSizeMsg{Width: 100, Height: 40})
	if m.macroH != 0 {
		t.Errorf("no macros → no bar row, got macroH=%d", m.macroH)
	}
	if m.renderMacroBar() != "" {
		t.Error("renderMacroBar should be empty with no macros")
	}
}
