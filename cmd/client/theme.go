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

// palette is a full named color scheme. applyTheme swaps every style over to one.
type palette struct {
	border, focus, title, room, exit, chatName, dim    lipgloss.Color
	hp, hpBg, xp, en, ok, bad, combat, player, enemy   lipgloss.Color
}

var palettes = map[string]palette{
	// The original dark, retro-terminal scheme.
	"default": {"240", "213", "117", "228", "84", "215", "244", "203", "236", "75", "179", "84", "203", "173", "81", "173"},
	// Monochrome amber, like an old CRT.
	"amber": {"94", "214", "214", "220", "178", "215", "94", "208", "235", "178", "214", "214", "208", "208", "222", "208"},
	// Green phosphor.
	"green": {"22", "46", "48", "84", "40", "120", "28", "46", "235", "84", "48", "46", "196", "84", "120", "84"},
	// Tuned for light terminals.
	"light": {"250", "161", "25", "94", "28", "130", "245", "124", "254", "26", "136", "28", "124", "130", "26", "130"},
}

func themeNames() []string { return []string{"default", "amber", "green", "light"} }

// applyTheme swaps the live palette and rebuilds every style from it. Unknown
// names fall back to "default". Safe to call at startup and on /theme.
func applyTheme(name string) {
	p, ok := palettes[name]
	if !ok {
		p = palettes["default"]
	}
	colBorder, colFocus, colTitle, colRoom, colExit, colChatName, colDim =
		p.border, p.focus, p.title, p.room, p.exit, p.chatName, p.dim
	colHP, colHPBg, colXP, colEN, colOK, colBad = p.hp, p.hpBg, p.xp, p.en, p.ok, p.bad

	titleStyle = lipgloss.NewStyle().Foreground(p.title).Bold(true)
	roomNameStyle = lipgloss.NewStyle().Foreground(p.room).Bold(true)
	exitStyle = lipgloss.NewStyle().Foreground(p.exit)
	dimStyle = lipgloss.NewStyle().Foreground(p.dim)
	chatNameStyle = lipgloss.NewStyle().Foreground(p.chatName).Bold(true)
	okStyle = lipgloss.NewStyle().Foreground(p.ok).Bold(true)
	badStyle = lipgloss.NewStyle().Foreground(p.bad).Bold(true)
	hpFillStyle = lipgloss.NewStyle().Foreground(p.hp)
	hpTroughStyle = lipgloss.NewStyle().Foreground(p.hpBg)
	xpFillStyle = lipgloss.NewStyle().Foreground(p.xp)
	enFillStyle = lipgloss.NewStyle().Foreground(p.en)
	noticeStyle = lipgloss.NewStyle().Foreground(p.title).Bold(true)
	combatStyle = lipgloss.NewStyle().Foreground(p.combat)
	playerStyle = lipgloss.NewStyle().Foreground(p.player).Bold(true)
	enemyFillStyle = lipgloss.NewStyle().Foreground(p.enemy)
	commsActiveStyle = lipgloss.NewStyle().Foreground(p.title).Bold(true).Underline(true)
	commsUnreadStyle = lipgloss.NewStyle().Foreground(p.chatName).Bold(true)
	macroKeyStyle = lipgloss.NewStyle().Foreground(p.exit).Bold(true)
}

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
