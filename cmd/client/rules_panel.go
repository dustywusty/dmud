package main

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// ruleEntry is one row in the RULES overlay: an alias, highlight, or trigger.
type ruleEntry struct {
	kind  string // "alias" | "highlight" | "trigger"
	name  string // alias name (delete key)
	index int    // index within highlights/triggers (delete key)
	label string // display text
	edit  string // command loaded into the input when editing
}

// ruleEntries flattens the configured aliases, highlights, and triggers into a
// stable, navigable list.
func (m model) ruleEntries() []ruleEntry {
	var es []ruleEntry

	for _, slot := range m.macroOrder() {
		es = append(es, ruleEntry{
			kind: "macro", name: slot,
			label: macroKeyStyle.Render(slotKeyLabel(slot)) + dimStyle.Render("  ") + macroRowText(m.macros[slot]),
			edit:  "/macro edit " + slot, // 'e' opens the editor; this is the typed equivalent
		})
	}

	names := make([]string, 0, len(m.aliases))
	for n := range m.aliases {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		exp := m.aliases[n]
		es = append(es, ruleEntry{
			kind: "alias", name: n,
			label: chatNameStyle.Render(n) + dimStyle.Render(" → ") + exp,
			edit:  "/alias " + n + " " + exp,
		})
	}
	for i, h := range m.highlights {
		es = append(es, ruleEntry{
			kind: "highlight", index: i,
			label: dimStyle.Render("/") + h.Pattern + dimStyle.Render("/  ") + h.Value,
			edit:  "/highlight " + h.Value + " " + h.Pattern,
		})
	}
	for i, t := range m.triggers {
		es = append(es, ruleEntry{
			kind: "trigger", index: i,
			label: dimStyle.Render("/") + t.Pattern + dimStyle.Render("/ → ") + t.Value,
			edit:  "/trigger " + t.Pattern + " = " + t.Value,
		})
	}
	return es
}

func (m *model) deleteSelectedRule() tea.Cmd {
	es := m.ruleEntries()
	if m.rulesCursor < 0 || m.rulesCursor >= len(es) {
		return nil
	}
	e := es[m.rulesCursor]
	switch e.kind {
	case "macro":
		delete(m.macros, e.name)
		m.recalc() // the bar may disappear
	case "alias":
		delete(m.aliases, e.name)
	case "highlight":
		m.highlights = append(m.highlights[:e.index], m.highlights[e.index+1:]...)
	case "trigger":
		m.triggers = append(m.triggers[:e.index], m.triggers[e.index+1:]...)
	}
	if n := len(m.ruleEntries()); m.rulesCursor >= n {
		m.rulesCursor = max(0, n-1)
	}
	return saveConfigCmd(m.config())
}

func (m model) handleRulesKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		_ = saveConfig(m.config())
		_ = saveMapData(mapFilePath(m.conn.addr), m.mapper.snapshot())
		m.conn.Close()
		return m, tea.Quit
	case "esc", "ctrl+a", "q":
		m.rulesOpen = false
		return m, nil
	case "up", "k":
		if m.rulesCursor > 0 {
			m.rulesCursor--
		}
		return m, nil
	case "down", "j":
		if m.rulesCursor < len(m.ruleEntries())-1 {
			m.rulesCursor++
		}
		return m, nil
	case "d", "delete":
		return m, m.deleteSelectedRule()
	case "e", "enter":
		es := m.ruleEntries()
		if m.rulesCursor >= 0 && m.rulesCursor < len(es) {
			e := es[m.rulesCursor]
			if e.kind == "macro" { // macros get the real multi-line editor
				m.rulesOpen = false
				return m, m.openMacroEditor(e.name)
			}
			m.in.SetValue(e.edit) // others: load into the command line to edit
			m.in.CursorEnd()
			m.rulesOpen = false
		}
		return m, nil
	}
	return m, nil
}

func kindLabel(kind string) string {
	switch kind {
	case "macro":
		return "MACROS"
	case "alias":
		return "ALIASES"
	case "highlight":
		return "HIGHLIGHTS"
	case "trigger":
		return "TRIGGERS"
	}
	return strings.ToUpper(kind)
}

// renderRules draws the full-screen RULES overlay.
func (m model) renderRules() string {
	entries := m.ruleEntries()

	var b strings.Builder
	b.WriteString(titleStyle.Render("RULES") +
		dimStyle.Render("   ↑↓ select · e edit · d delete · /macro /alias /highlight /trigger to add · Esc close") + "\n\n")

	if len(entries) == 0 {
		b.WriteString(dimStyle.Render("nothing yet — add with /macro, /alias, /highlight, or /trigger"))
	} else {
		lastKind := ""
		for i, e := range entries {
			if e.kind != lastKind {
				if lastKind != "" {
					b.WriteByte('\n')
				}
				b.WriteString(titleStyle.Render(kindLabel(e.kind)) + "\n")
				lastKind = e.kind
			}
			marker, line := "  ", e.label
			if i == m.rulesCursor {
				marker = exitStyle.Render("▸ ")
				line = playerStyle.Render(ansi.Strip(e.label)) // recolor the selected row
			}
			b.WriteString(marker + line + "\n")
		}
	}

	body := clipLines(b.String(), m.midH-2)
	return panelStyle(true).Width(m.width - 2).Height(m.midH - 2).Render(body)
}
