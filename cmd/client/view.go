package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	if !m.ready {
		return "starting dmud-client…"
	}

	header := m.renderHeader()
	status := m.renderStatus()
	input := m.renderInput()

	var middle string
	switch {
	case m.macroEd != nil:
		middle = m.renderMacroEditor()
	case m.rulesOpen:
		middle = m.renderRules()
	case m.fullMap:
		middle = m.renderFullMap()
	default:
		middle = lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.renderLeft(),
			m.renderMain(),
			m.renderRight(),
		)
	}

	// Status bar sits at the very top, under the header; the macro legend (when
	// any macros are set) sits just above the input.
	rows := []string{header, status, middle}
	if bar := m.renderMacroBar(); bar != "" {
		rows = append(rows, bar)
	}
	rows = append(rows, input)
	out := lipgloss.JoinVertical(lipgloss.Left, rows...)
	// Clamp to the terminal so an oversized panel can never push the layout
	// past the screen edge.
	return lipgloss.NewStyle().MaxWidth(m.width).MaxHeight(m.height).Render(out)
}

func (m model) renderHeader() string {
	dot := badStyle.Render("○")
	state := badStyle.Render("offline")
	if m.connected {
		dot = okStyle.Render("●")
		state = okStyle.Render("connected")
	}

	room := m.room.Title
	if room == "" {
		room = m.status.Area
	}

	left := fmt.Sprintf(" %s %s %s %s %s %s",
		titleStyle.Render("dmud"),
		dimStyle.Render("·"),
		dimStyle.Render(m.conn.transport.String()+" "+m.conn.addr),
		dot, state,
		roomNameStyle.Render(room),
	)
	right := dimStyle.Render("Tab focus · ^G map · ^A rules · ^N/P chat · ^C quit ")
	if m.layoutMode {
		right = roomNameStyle.Render("LAYOUT [" + m.focusName() + "] · Tab pane · ←→ ↑↓ resize · Esc done ")
	}
	if m.rulesOpen {
		right = roomNameStyle.Render("RULES · ↑↓ select · e edit · d delete · Esc close ")
	}

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	line := left + strings.Repeat(" ", gap) + right
	return lipgloss.NewStyle().MaxWidth(m.width).Render(line)
}

// renderLeft is the HERE panel: who and what is in the current room. (Room
// name and exits live on the MAP now.)
func (m model) focusName() string {
	switch m.focus {
	case focusHere:
		return "HERE"
	case focusMap:
		return "MAP"
	case focusChat:
		return "COMMS"
	default:
		return "OUTPUT"
	}
}

func (m model) renderLeft() string {
	innerH := m.midH - 2
	body := titleStyle.Render("HERE") + "\n" + m.here.View()
	return panelStyle(m.focus == focusHere).
		Width(m.colLeft - 2).
		Height(innerH).
		Render(body)
}

func (m model) renderMain() string {
	innerH := m.midH - 2
	body := titleStyle.Render("OUTPUT") + "\n" + m.main.View()
	return panelStyle(m.focus == focusMain).
		Width(m.colMain - 2).
		Height(innerH).
		Render(body)
}

// renderRight stacks the MAP (top) and COMMS (bottom) panels in the right
// column. Their heights sum to midH so the column lines up with MAIN.
func (m model) renderRight() string {
	if !m.showRight || m.rightW <= 0 {
		return ""
	}
	innerW := m.rightW - 2
	var boxes []string

	if m.mapH > 0 {
		body := m.mapHeader(innerW, "") + "\n" + m.mapper.render(innerW, m.mapH-4)
		boxes = append(boxes, panelStyle(m.focus == focusMap).
			Width(innerW).
			Height(m.mapH-2).
			Render(body))
	}

	commsH := m.midH - m.mapH
	body := clipWidth(m.comms.tabBar(), innerW) + "\n" + m.comms.vp.View()
	boxes = append(boxes, panelStyle(m.focus == focusChat).
		Width(innerW).
		Height(commsH-2).
		Render(body))

	return lipgloss.JoinVertical(lipgloss.Left, boxes...)
}

// renderFullMap fills the middle band (header/status/input stay put) with the map.
func (m model) renderFullMap() string {
	body := m.mapHeader(m.width-2, "Esc / ^G to close") + "\n" +
		m.mapper.render(m.width-2, m.midH-4)
	return panelStyle(true).Width(m.width - 2).Height(m.midH - 2).Render(body)
}

// mapHeader shows the current room's name and exits above the map graph.
func (m model) mapHeader(width int, hint string) string {
	name := dimStyle.Render("(unknown location)")
	if m.room.Title != "" {
		name = roomNameStyle.Render(m.room.Title)
	}
	exits := exitsSummary(m.room.Exits)
	if hint != "" {
		exits += dimStyle.Render("   " + hint)
	}
	return clipWidth(name, width) + "\n" + clipWidth(exits, width)
}

func exitsSummary(exits []string) string {
	if len(exits) == 0 {
		return dimStyle.Render("exits: none")
	}
	return dimStyle.Render("exits: ") + exitStyle.Render(strings.Join(exits, ", "))
}

func (m model) renderStatus() string {
	var line string
	if !m.hasGot {
		line = dimStyle.Render(" awaiting status…")
	} else {
		s := m.status
		segs := []string{
			" " + hpFillStyle.Render("HP") + " " + bar(s.HP, s.MaxHP, 12, hpFillStyle, hpTroughStyle) +
				fmt.Sprintf(" %d/%d", s.HP, s.MaxHP),
		}
		if s.HasEP {
			segs = append(segs, enFillStyle.Render("EN")+" "+bar(s.EP, s.MaxEP, 10, enFillStyle, hpTroughStyle)+
				fmt.Sprintf(" %d/%d", s.EP, s.MaxEP))
		}
		segs = append(segs,
			xpFillStyle.Render("XP")+" "+bar(s.XP, s.ReqXP, 10, xpFillStyle, hpTroughStyle)+
				fmt.Sprintf(" %d/%d", s.XP, s.ReqXP),
			fmt.Sprintf("Lv %d", s.Level),
		)
		if s.Gold > 0 {
			segs = append(segs, goldStyle.Render(fmt.Sprintf("%d gold", s.Gold)))
		}
		segs = append(segs, "FX "+fxLabel(s.Fx, s.Effects))
		if seg := statsSeg(s.Stats); seg != "" {
			segs = append(segs, seg)
		}
		if seg := targetSeg(m.target); seg != "" {
			segs = append(segs, seg)
		}
		line = strings.Join(segs, dimStyle.Render("   │   "))
	}

	return panelStyle(false).
		Width(m.width - 2).
		Height(1).
		Render(lipgloss.NewStyle().MaxWidth(m.width - 4).Render(line))
}

func (m model) renderInput() string {
	return panelStyle(false).
		Width(m.width - 2).
		Height(1).
		Render(m.in.View())
}

// --- small render helpers ---

func effectsLabel(effects []string) string {
	if len(effects) == 0 {
		return dimStyle.Render("none")
	}
	return strings.Join(effects, ", ")
}

var (
	goldStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("220"))
	fxDotStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("203")) // damage-over-time
	fxCtrlStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214")) // control
	fxHealStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("84"))  // heal-over-time
	fxBuffStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("117")) // buff
)

// fxLabel renders active effects with countdowns, colored by kind. Falls back to
// the legacy name-only list when the server doesn't send structured fx.
func fxLabel(fx []fxEffect, legacy []string) string {
	if len(fx) == 0 {
		return effectsLabel(legacy)
	}
	parts := make([]string, 0, len(fx))
	for _, f := range fx {
		st := fxBuffStyle
		switch f.Kind {
		case "dot":
			st = fxDotStyle
		case "control":
			st = fxCtrlStyle
		case "heal":
			st = fxHealStyle
		}
		label := f.Name
		if f.Remaining > 0 {
			label += fmt.Sprintf(" %ds", f.Remaining)
		}
		parts = append(parts, st.Render(label))
	}
	return strings.Join(parts, ", ")
}

// statsSeg renders the five attributes compactly for the status line, e.g.
// "STR20 DEX4 CON20 INT2 WIS6". Empty when the server sends no stats.
func statsSeg(stats map[string]int) string {
	if len(stats) == 0 {
		return ""
	}
	parts := make([]string, 0, 5)
	for _, k := range []string{"STR", "DEX", "CON", "INT", "WIS"} {
		if v, ok := stats[k]; ok {
			parts = append(parts, dimStyle.Render(k)+fmt.Sprintf("%d", v))
		}
	}
	return strings.Join(parts, " ")
}

// targetSeg renders the current attack target as an enemy HP bar for the status
// line, or "" when there is no target. A slain target lingers (marked) until the
// player moves or engages something else.
func targetSeg(t combatMsg) string {
	if t.Target == "" {
		return ""
	}
	if t.Killed || t.TargetHP <= 0 {
		return dimStyle.Render("† " + t.Target + " slain")
	}
	return combatStyle.Render("⚔ "+t.Target) + " " +
		bar(t.TargetHP, t.TargetMax, 10, enemyFillStyle, hpTroughStyle) +
		fmt.Sprintf(" %d/%d", t.TargetHP, t.TargetMax)
}

// hereList renders room occupants: confirmed players are highlighted, corpses
// are dimmed with a marker, and everything else (NPCs, items) is listed plainly.
func hereList(occupants []string, roster map[string]bool) string {
	if len(occupants) == 0 {
		return dimStyle.Render("(empty)")
	}
	var b strings.Builder
	for i, name := range occupants {
		if i > 0 {
			b.WriteByte('\n')
		}
		switch {
		case roster[name]:
			b.WriteString(playerStyle.Render("◆ " + name))
		case strings.Contains(strings.ToLower(name), "corpse"):
			b.WriteString(dimStyle.Render("† " + name))
		default:
			b.WriteString(dimStyle.Render("· ") + name)
		}
	}
	return b.String()
}

// clipLines keeps at most n lines so a panel's body never exceeds its height.
func clipLines(s string, n int) string {
	if n < 1 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

// wrap soft-wraps content to a given width for display inside a viewport.
func wrap(s string, w int) string {
	if w < 1 {
		return s
	}
	return lipgloss.NewStyle().Width(w).Render(s)
}

// colorizeChat timestamps a chat line and highlights the speaker's name.
func colorizeChat(s string) string {
	ts := dimStyle.Render(time.Now().Format("15:04"))
	if i := strings.IndexByte(s, ':'); i > 0 {
		return ts + " " + chatNameStyle.Render(s[:i]) + s[i:]
	}
	return ts + " " + s
}
