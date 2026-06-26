package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// macroEditor is the full-screen overlay for editing one macro: a single-line
// label field plus a multi-line command body (one command per line). It's the
// "real editor" path; the /macro command stays for quick single-line binds.
type macroEditor struct {
	slot  string          // "f1".."f12"
	label textinput.Model // optional label
	body  textarea.Model  // command body, one per line
	focus int             // 0 = label, 1 = body
}

// openMacroEditor opens the editor for a slot, preloading any existing macro.
// Returns the focus command (cursor blink) for the active field.
func (m *model) openMacroEditor(slot string) tea.Cmd {
	la := textinput.New()
	la.Prompt = ""
	la.CharLimit = 64
	la.Placeholder = "(optional label, shown on the bar)"

	ta := textarea.New()
	ta.Placeholder = "one command per line — e.g.\ncast heal on self\nquaff potion"
	ta.ShowLineNumbers = true
	ta.CharLimit = 0 // no limit

	if mac, ok := m.macros[slot]; ok {
		la.SetValue(mac.Label)
		ta.SetValue(mac.Cmd)
	}

	ed := &macroEditor{slot: slot, label: la, body: ta, focus: 1}
	m.macroEd = ed
	m.sizeMacroEditor()
	return ed.syncFocus()
}

// syncFocus focuses the active field and blurs the other, returning the focused
// field's cursor-blink command.
func (e *macroEditor) syncFocus() tea.Cmd {
	if e.focus == 0 {
		e.body.Blur()
		return e.label.Focus()
	}
	e.label.Blur()
	return e.body.Focus()
}

func (e *macroEditor) toggleFocus() tea.Cmd {
	e.focus = 1 - e.focus
	return e.syncFocus()
}

// sizeMacroEditor fits the editor's fields to the current middle band.
func (m *model) sizeMacroEditor() {
	if m.macroEd == nil || !m.ready {
		return
	}
	innerW := max(10, m.width-6)
	m.macroEd.label.Width = max(8, innerW-9)
	m.macroEd.body.SetWidth(innerW)
	// Reserve rows for the title, label, body caption, and hint lines.
	m.macroEd.body.SetHeight(max(3, m.midH-7))
}

// handleMacroEditorKey drives the editor overlay. Esc cancels, ^S saves, Tab
// switches fields; everything else goes to the focused field (Enter inserts a
// newline in the body, so multi-line bodies just work).
func (m model) handleMacroEditorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		_ = saveConfig(m.config())
		_ = saveMapData(mapFilePath(m.conn.addr), m.mapper.snapshot())
		m.conn.Close()
		return m, tea.Quit
	case "esc":
		m.macroEd = nil
		return m, nil
	case "ctrl+s":
		return m, m.saveMacroEditor()
	case "tab", "shift+tab":
		return m, m.macroEd.toggleFocus()
	case "enter":
		if m.macroEd.focus == 0 { // from the label, Enter drops into the body
			m.macroEd.focus = 1
			return m, m.macroEd.syncFocus()
		}
	}

	var cmd tea.Cmd
	if m.macroEd.focus == 0 {
		m.macroEd.label, cmd = m.macroEd.label.Update(msg)
	} else {
		m.macroEd.body, cmd = m.macroEd.body.Update(msg)
	}
	return m, cmd
}

// saveMacroEditor stores the edited macro (or clears the slot if the body is
// empty) and closes the overlay.
func (m *model) saveMacroEditor() tea.Cmd {
	ed := m.macroEd
	m.macroEd = nil

	cmd := cleanMacroBody(ed.body.Value())
	if cmd == "" {
		if _, ok := m.macros[ed.slot]; ok {
			delete(m.macros, ed.slot)
			m.appendMain(noticeStyle.Render("macro cleared: " + slotKeyLabel(ed.slot)))
		}
		m.recalc()
		return saveConfigCmd(m.config())
	}

	if m.macros == nil {
		m.macros = map[string]macro{}
	}
	m.macros[ed.slot] = macro{Label: strings.TrimSpace(ed.label.Value()), Cmd: cmd}
	m.appendMain(noticeStyle.Render("macro saved: " + slotKeyLabel(ed.slot)))
	m.recalc()
	return saveConfigCmd(m.config())
}

// cleanMacroBody trims trailing whitespace on each line and drops blank lines at
// the start and end, leaving interior blanks intact.
func cleanMacroBody(s string) string {
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		lines[i] = strings.TrimRight(ln, " \t")
	}
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return strings.Join(lines, "\n")
}

// renderMacroEditor draws the editor overlay into the middle band.
func (m model) renderMacroEditor() string {
	ed := m.macroEd

	labelMark, bodyMark := "  ", "  "
	if ed.focus == 0 {
		labelMark = exitStyle.Render("▸ ")
	} else {
		bodyMark = exitStyle.Render("▸ ")
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("MACRO "+slotKeyLabel(ed.slot)) +
		dimStyle.Render("   ^S save · Esc cancel · Tab switch field · Enter = new line in body") + "\n\n")
	b.WriteString(labelMark + dimStyle.Render("label    ") + ed.label.View() + "\n\n")
	b.WriteString(bodyMark + dimStyle.Render("commands (one per line; ; and aliases also work)") + "\n")
	b.WriteString(ed.body.View())

	body := clipLines(b.String(), m.midH-2)
	return panelStyle(true).Width(m.width - 2).Height(m.midH - 2).Render(body)
}
