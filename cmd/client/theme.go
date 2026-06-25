package main

import "github.com/charmbracelet/lipgloss"

// Palette — a dark, retro-terminal scheme echoing the classic MUD clients.
var (
	colBorder   = lipgloss.Color("240") // dim grey, unfocused panel
	colFocus    = lipgloss.Color("213") // magenta, focused panel
	colTitle    = lipgloss.Color("117") // cyan, panel titles
	colRoom     = lipgloss.Color("228") // yellow, room name
	colExit     = lipgloss.Color("84")  // green, exit chips
	colChatName = lipgloss.Color("215") // orange, speaker names
	colDim      = lipgloss.Color("244") // muted text
	colHP       = lipgloss.Color("203") // red, health
	colHPBg     = lipgloss.Color("236") // bar trough
	colXP       = lipgloss.Color("75")  // blue, experience
	colEN       = lipgloss.Color("179") // amber, endurance/stamina
	colOK       = lipgloss.Color("84")  // green, connected
	colBad      = lipgloss.Color("203") // red, disconnected
)

// panelStyle returns a bordered box; the border brightens when focused.
func panelStyle(focused bool) lipgloss.Style {
	c := colBorder
	if focused {
		c = colFocus
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(c)
}

var (
	titleStyle    = lipgloss.NewStyle().Foreground(colTitle).Bold(true)
	roomNameStyle = lipgloss.NewStyle().Foreground(colRoom).Bold(true)
	exitStyle     = lipgloss.NewStyle().Foreground(colExit)
	dimStyle      = lipgloss.NewStyle().Foreground(colDim)
	chatNameStyle = lipgloss.NewStyle().Foreground(colChatName).Bold(true)
	okStyle       = lipgloss.NewStyle().Foreground(colOK).Bold(true)
	badStyle      = lipgloss.NewStyle().Foreground(colBad).Bold(true)
	hpFillStyle   = lipgloss.NewStyle().Foreground(colHP)
	hpTroughStyle = lipgloss.NewStyle().Foreground(colHPBg)
	xpFillStyle   = lipgloss.NewStyle().Foreground(colXP)
	enFillStyle   = lipgloss.NewStyle().Foreground(colEN)

	noticeStyle    = lipgloss.NewStyle().Foreground(colTitle).Bold(true)             // STATUS| notifications
	combatStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("173"))           // DMG| combat
	playerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("81")).Bold(true) // online players in HERE
	enemyFillStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("173"))           // target HP bar

	commsActiveStyle = lipgloss.NewStyle().Foreground(colTitle).Bold(true).Underline(true) // active COMMS tab
	commsUnreadStyle = lipgloss.NewStyle().Foreground(colChatName).Bold(true)              // COMMS tab with unread
	macroKeyStyle    = lipgloss.NewStyle().Foreground(colExit).Bold(true)                  // F-key labels on the macro bar
)

// bar renders a fixed-width [####----] progress bar.
func bar(cur, max, width int, fill, trough lipgloss.Style) string {
	if width < 1 {
		return ""
	}
	ratio := 0.0
	if max > 0 {
		ratio = float64(cur) / float64(max)
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	filled := int(ratio * float64(width))
	out := ""
	if filled > 0 {
		out += fill.Render(repeat('█', filled))
	}
	if width-filled > 0 {
		out += trough.Render(repeat('░', width-filled))
	}
	return out
}

func repeat(r rune, n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]rune, n)
	for i := range b {
		b[i] = r
	}
	return string(b)
}
