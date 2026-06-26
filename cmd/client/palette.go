package main

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// cmdPalette is the Ctrl+K fuzzy command finder overlay: type to filter the
// command vocabulary (plus your aliases and macro labels), ↑↓ to pick, Enter to
// drop it into the input line ready for arguments.
type cmdPalette struct {
	query    string
	all      []string
	filtered []string
	cursor   int
}

func (m *model) openPalette() {
	cands := append([]string{}, baseCommands...)
	for a := range m.aliases {
		cands = append(cands, a)
	}
	for _, mc := range m.macros {
		if mc.Label != "" {
			cands = append(cands, mc.Label)
		}
	}
	p := &cmdPalette{all: dedupeSorted(cands)}
	p.refilter()
	m.cmdpal = p
}

func dedupeSorted(s []string) []string {
	sort.Strings(s)
	out := s[:0]
	var last string
	for i, v := range s {
		if i == 0 || v != last {
			out = append(out, v)
			last = v
		}
	}
	return out
}

// fuzzyMatch reports whether query is a (case-insensitive) subsequence of target.
func fuzzyMatch(query, target string) bool {
	if query == "" {
		return true
	}
	q, t := strings.ToLower(query), strings.ToLower(target)
	qi := 0
	for ti := 0; ti < len(t) && qi < len(q); ti++ {
		if t[ti] == q[qi] {
			qi++
		}
	}
	return qi == len(q)
}

func (p *cmdPalette) refilter() {
	var out []string
	for _, c := range p.all {
		if fuzzyMatch(p.query, c) {
			out = append(out, c)
		}
	}
	q := strings.ToLower(p.query)
	sort.SliceStable(out, func(i, j int) bool {
		pi := strings.HasPrefix(strings.ToLower(out[i]), q)
		pj := strings.HasPrefix(strings.ToLower(out[j]), q)
		if pi != pj {
			return pi // prefix matches rank first
		}
		return len(out[i]) < len(out[j]) // then the tightest match
	})
	p.filtered = out
	if p.cursor >= len(out) {
		p.cursor = 0
	}
}

func (m model) handlePaletteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	p := m.cmdpal
	switch msg.String() {
	case "esc", "ctrl+k", "ctrl+c":
		m.cmdpal = nil
	case "enter":
		if p.cursor < len(p.filtered) {
			m.in.SetValue(p.filtered[p.cursor] + " ")
			m.in.CursorEnd()
		}
		m.cmdpal = nil
	case "up", "ctrl+p":
		if p.cursor > 0 {
			p.cursor--
		}
	case "down", "ctrl+n":
		if p.cursor < len(p.filtered)-1 {
			p.cursor++
		}
	case "backspace":
		if p.query != "" {
			p.query = p.query[:len(p.query)-1]
			p.refilter()
		}
	default:
		if len(msg.Runes) == 1 {
			p.query += string(msg.Runes)
			p.refilter()
		}
	}
	return m, nil
}

func (m model) renderPalette() string {
	p := m.cmdpal
	var b strings.Builder
	b.WriteString(titleStyle.Render("⌘ command palette") + "   " +
		dimStyle.Render("type to filter · ↑↓ pick · ⏎ insert · esc") + "\n")
	b.WriteString("› " + p.query + dimStyle.Render("▏") + "\n\n")

	const maxRows = 12
	if len(p.filtered) == 0 {
		b.WriteString(dimStyle.Render("  (no matches)"))
	}
	for i, c := range p.filtered {
		if i >= maxRows {
			b.WriteString(dimStyle.Render(fmt.Sprintf("  … and %d more", len(p.filtered)-maxRows)))
			break
		}
		if i == p.cursor {
			b.WriteString(commsActiveStyle.Render("▸ "+c) + "\n")
		} else {
			b.WriteString("  " + c + "\n")
		}
	}
	return panelStyle(true).Width(m.width - 2).Render(b.String())
}
