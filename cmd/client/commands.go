package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// baseCommands seeds tab-completion with the server's command vocabulary
// (including the wylie content verbs).
var baseCommands = []string{
	"look", "who", "exit", "name", "login", "save", "summon", "cast", "recall",
	"say", "shout", "kill", "examine", "time", "history", "clear", "suggest",
	"complete", "help", "loot", "inventory", "get", "drop", "dropall",
	"sacrifice", "sacall", "hail", "uptime",
	"eat", "drink", "quaff", "consume",
	"spells", "spellbook", "stats", "score", "race", "classes", "class",
	"spark", "firebolt", "scorch", "fireball", "immolate",
	"mend", "heal", "greater heal", "moonfire", "moonbeam", "starfall",
	"smite", "bless", "shadow bolt", "drain",
	"human", "dwarf", "elf", "goblin", "ogre",
	"shift", "shapeshift", "revert", "bear", "wolf", "panther",
	"north", "south", "east", "west", "up", "down",
	"give", "buy", "list", "faction", "mount", "ride", "dismount",
}

// submit handles a line of user input: a /slash command runs locally, otherwise
// it is echoed, recorded in history, and expanded into server commands.
func (m *model) submit(raw string) tea.Cmd {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	// Record in history (skip consecutive duplicates) and reset browsing state.
	if len(m.history) == 0 || m.history[len(m.history)-1] != raw {
		m.history = append(m.history, raw)
	}
	m.histPos = len(m.history)
	m.draft = ""

	var cmds []tea.Cmd
	if c := appendHistoryCmd(raw); c != nil {
		cmds = append(cmds, c)
	}

	if strings.HasPrefix(raw, "/") {
		cmds = append(cmds, m.runSlash(raw))
		return tea.Batch(cmds...)
	}

	m.appendMain(dimStyle.Render("» ") + raw)
	for _, c := range m.expand(raw) {
		if c == "exit" {
			m.quitting = true
		}
		cmds = append(cmds, m.dispatch(c))
	}
	return tea.Batch(cmds...)
}

// dispatch routes one already-expanded command: /slash runs locally, anything
// else is sent to the server (and noted for the auto-mapper).
func (m *model) dispatch(cmd string) tea.Cmd {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil
	}
	if strings.HasPrefix(cmd, "/") {
		return m.runSlash(cmd)
	}
	m.mapper.noteCommand(cmd)
	return sendCmd(m.conn, cmd)
}

// fireTriggers runs trigger rules against a displayed line and returns the
// resulting commands.
func (m *model) fireTriggers(display string) []tea.Cmd {
	var cmds []tea.Cmd
	for _, c := range matchTriggers(ansi.Strip(display), m.triggers) {
		cmds = append(cmds, m.dispatch(c))
	}
	return cmds
}

// expand applies the alias on the first word (which may itself hold several
// ;-separated commands) and speedwalk expansion to each result.
func (m *model) expand(line string) []string {
	if fields := strings.Fields(line); len(fields) > 0 {
		if exp, ok := m.aliases[strings.ToLower(fields[0])]; ok {
			rest := strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
			if rest != "" {
				line = exp + " " + rest
			} else {
				line = exp
			}
		}
	}

	var cmds []string
	for _, part := range strings.Split(line, ";") {
		if part = strings.TrimSpace(part); part != "" {
			cmds = append(cmds, expandSpeedwalk(part)...)
		}
	}
	return cmds
}

var speedwalkRe = regexp.MustCompile(`^(?:[0-9]*[nsewud])+$`)

// expandSpeedwalk turns "3n2e" into [n n n e e]. A single direction or any
// non-speedwalk token is returned unchanged.
func expandSpeedwalk(cmd string) []string {
	low := strings.ToLower(cmd)
	if len(low) < 2 || !speedwalkRe.MatchString(low) {
		return []string{cmd}
	}
	var out []string
	count := 0
	for _, r := range low {
		if r >= '0' && r <= '9' {
			count = count*10 + int(r-'0')
			continue
		}
		n := count
		if n == 0 {
			n = 1
		}
		for i := 0; i < n; i++ {
			out = append(out, string(r))
		}
		count = 0
	}
	return out
}

// completeInput completes the last whitespace-delimited token of the input
// against known commands, exits, players, and room occupants.
func (m *model) completeInput() {
	val := m.in.Value()
	prefix, token := "", val
	if i := strings.LastIndex(val, " "); i >= 0 {
		prefix, token = val[:i+1], val[i+1:]
	}
	if token == "" {
		return
	}

	lt := strings.ToLower(token)
	var matches []string
	for _, c := range m.candidates() {
		if strings.HasPrefix(strings.ToLower(c), lt) {
			matches = append(matches, c)
		}
	}
	if len(matches) == 0 {
		return
	}

	completion := matches[0]
	if len(matches) > 1 {
		if cp := longestCommonPrefix(matches); len(cp) > len(token) {
			completion = cp
		} else {
			completion = token // ambiguous, nothing more to add
		}
	}
	m.in.SetValue(prefix + completion)
	m.in.CursorEnd()
}

func (m *model) candidates() []string {
	c := append([]string{}, baseCommands...)
	for name := range m.roster {
		c = append(c, name)
	}
	for _, occ := range m.room.Occupants {
		c = append(c, occ)
		for _, w := range strings.Fields(occ) {
			if len(w) >= 3 {
				c = append(c, w)
			}
		}
	}
	c = append(c, m.room.Exits...)
	return c
}

func longestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	prefix := strs[0]
	for _, s := range strs[1:] {
		for !strings.HasPrefix(strings.ToLower(s), strings.ToLower(prefix)) {
			prefix = prefix[:len(prefix)-1]
			if prefix == "" {
				return ""
			}
		}
	}
	return prefix
}

// --- /slash commands ---

func (m *model) runSlash(line string) tea.Cmd {
	body := strings.TrimSpace(strings.TrimPrefix(line, "/"))
	cmd, rest, _ := strings.Cut(body, " ")
	cmd = strings.ToLower(cmd)
	rest = strings.TrimSpace(rest)

	switch cmd {
	case "help", "?":
		m.appendMain(slashHelp())
		return nil
	case "quit":
		m.quitting = true
		_ = saveConfig(m.config())
		_ = saveMapData(mapFilePath(m.conn.addr), m.mapper.snapshot())
		m.conn.Close()
		return tea.Quit
	case "clear":
		m.mainRaw = ""
		m.main.SetContent("")
		return nil
	case "reconnect":
		return m.reconnectNow()
	case "spacing":
		m.compact = !m.compact
		m.comms.roomy = !m.compact
		if m.compact {
			m.appendMain(noticeStyle.Render("compact spacing (no blank lines)"))
		} else {
			m.appendMain(noticeStyle.Render("roomy spacing (blank line between messages)"))
		}
		return saveConfigCmd(m.config())
	case "map":
		if strings.EqualFold(rest, "reset") {
			m.mapper = newMapper()
			_ = os.Remove(mapFilePath(m.conn.addr))
			if m.room.Title != "" {
				m.mapper.arrive(m.room.Title, m.room.Exits) // re-seed current room
			}
			m.appendMain(noticeStyle.Render("map reset"))
			return nil
		}
		m.fullMap = !m.fullMap
		return nil
	case "resize", "layout":
		m.layoutMode = true
		if m.focus == focusMain {
			m.cycleFocus(1)
		}
		return nil
	case "keys":
		m.keyDebug = !m.keyDebug
		if m.keyDebug {
			m.appendMain(noticeStyle.Render("key debug ON — press keys to see their names; /keys to stop"))
		} else {
			m.appendMain(noticeStyle.Render("key debug OFF"))
		}
		return nil
	case "bell":
		return bellCmd()
	case "mouse":
		m.mouseOn = !m.mouseOn
		if m.mouseOn {
			m.appendMain(noticeStyle.Render("mouse on — click panels/tabs, wheel scrolls"))
			return tea.EnableMouseCellMotion
		}
		m.appendMain(noticeStyle.Render("mouse off — use your terminal to select/copy text"))
		return tea.DisableMouse
	case "find", "search":
		m.findInOutput(rest)
		return nil
	case "log":
		return m.toggleLog(rest)
	case "macro", "mac":
		return m.handleMacro(rest)
	case "alias":
		return m.handleAlias(rest)
	case "highlight", "hl":
		return m.handleHighlight(rest)
	case "trigger", "trig":
		return m.handleTrigger(rest)
	case "alarm":
		return m.handleAlarm(rest)
	case "theme":
		return m.handleTheme(rest)
	case "prompt":
		return m.handlePrompt(rest)
	default:
		m.appendMain(dimStyle.Render("unknown command: /" + cmd + "   (try /help)"))
		return nil
	}
}

func slashHelp() string {
	lines := []string{
		titleStyle.Render("client commands"),
		"  /help                 this list",
		"  /quit                 leave the client",
		"  /clear                clear the OUTPUT panel",
		"  /reconnect            drop and redial the server",
		"  /resize               resize panels with arrow keys (or press ^O); Esc done",
		"  /map [reset]          toggle the full-screen map (reset wipes the saved map)",
		"  /spacing              toggle blank lines between messages",
		"  /find <text>          jump OUTPUT to the latest match",
		"  /log [path]           start/stop logging the session",
		"  /keys                 echo key names (to debug bindings)",
		"  /mouse                toggle mouse capture (off lets you select/copy text)",
		"  /macro <1-12> [label =] <cmd>     bind a hotkey: Shift+digit / F-key (/macro 1 to clear)",
		"  /macro edit <1-12>               open the multi-line editor (^S save · Esc cancel)",
		"  /alias [name [cmds]]  list / set / remove an alias (cmds joined by ;)",
		"  /highlight <color> <regex>   colorize matching output (/highlight off)",
		"  /trigger <regex> = <cmd>     auto-run a command on match (/trigger off)",
		"  /bell                 ring the terminal bell",
		dimStyle.Render("  ⇧1-0 / F-key macros · ^G map · ^A rules · ^O resize · speedwalk 3n2e · Tab complete"),
	}
	return strings.Join(lines, "\n")
}

func (m *model) handleAlias(rest string) tea.Cmd {
	if rest == "" {
		if len(m.aliases) == 0 {
			m.appendMain(dimStyle.Render("no aliases set"))
			return nil
		}
		var b strings.Builder
		b.WriteString(titleStyle.Render("aliases"))
		for name, exp := range m.aliases {
			b.WriteString("\n  " + chatNameStyle.Render(name) + " → " + exp)
		}
		m.appendMain(b.String())
		return nil
	}
	name, exp, _ := strings.Cut(rest, " ")
	name = strings.ToLower(strings.TrimSpace(name))
	exp = strings.TrimSpace(exp)
	if exp == "" {
		delete(m.aliases, name)
		m.appendMain(noticeStyle.Render("alias removed: " + name))
	} else {
		m.aliases[name] = exp
		m.appendMain(noticeStyle.Render("alias set: " + name + " → " + exp))
	}
	return saveConfigCmd(m.config())
}

func (m *model) handleHighlight(rest string) tea.Cmd {
	switch {
	case rest == "":
		m.appendMain(listRules("highlights", m.highlights))
		return nil
	case rest == "off":
		m.highlights = nil
		m.appendMain(noticeStyle.Render("highlights cleared"))
		return saveConfigCmd(m.config())
	}
	color, pattern, _ := strings.Cut(rest, " ")
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		m.appendMain(badStyle.Render("usage: /highlight <color> <regex>"))
		return nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		m.appendMain(badStyle.Render("bad pattern: " + err.Error()))
		return nil
	}
	m.highlights = append(m.highlights, rule{Pattern: pattern, Value: colorCode(color), re: re})
	m.appendMain(noticeStyle.Render(fmt.Sprintf("highlight added: /%s/ in %s", pattern, color)))
	return saveConfigCmd(m.config())
}

func (m *model) handleTrigger(rest string) tea.Cmd {
	switch {
	case rest == "":
		m.appendMain(listRules("triggers", m.triggers))
		return nil
	case rest == "off":
		m.triggers = nil
		m.appendMain(noticeStyle.Render("triggers cleared"))
		return saveConfigCmd(m.config())
	}
	pattern, send, ok := strings.Cut(rest, " = ")
	pattern, send = strings.TrimSpace(pattern), strings.TrimSpace(send)
	if !ok || pattern == "" || send == "" {
		m.appendMain(badStyle.Render("usage: /trigger <regex> = <command>"))
		return nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		m.appendMain(badStyle.Render("bad pattern: " + err.Error()))
		return nil
	}
	m.triggers = append(m.triggers, rule{Pattern: pattern, Value: send, re: re})
	m.appendMain(noticeStyle.Render(fmt.Sprintf("trigger added: /%s/ → %s", pattern, send)))
	return saveConfigCmd(m.config())
}

func listRules(kind string, rules []rule) string {
	if len(rules) == 0 {
		return dimStyle.Render("no " + kind + " set")
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(kind))
	for _, r := range rules {
		b.WriteString("\n  /" + r.Pattern + "/ → " + r.Value)
	}
	return b.String()
}

func (m *model) findInOutput(text string) {
	if text == "" {
		return
	}
	lines := strings.Split(wrap(m.mainRaw, m.main.Width), "\n")
	needle := strings.ToLower(text)
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(strings.ToLower(ansi.Strip(lines[i])), needle) {
			m.main.SetYOffset(i)
			return
		}
	}
	m.appendMain(dimStyle.Render("/find: no match for \"" + text + "\""))
}

func (m *model) toggleLog(arg string) tea.Cmd {
	if m.logFile != nil {
		_ = m.logFile.Close()
		m.logFile = nil
		m.appendMain(noticeStyle.Render("logging stopped"))
		return nil
	}
	path := arg
	if path == "" {
		dir := filepath.Join(configDir(), "logs")
		_ = os.MkdirAll(dir, 0o755)
		path = filepath.Join(dir, "session-"+time.Now().Format("20060102-150405")+".log")
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		m.appendMain(badStyle.Render("log open failed: " + err.Error()))
		return nil
	}
	m.logFile = f
	m.appendMain(noticeStyle.Render("logging to " + path))
	return nil
}

// handleTheme views or switches the color theme.
func (m *model) handleTheme(rest string) tea.Cmd {
	rest = strings.TrimSpace(strings.ToLower(rest))
	if rest == "" {
		cur := m.themeName
		if cur == "" {
			cur = "default"
		}
		m.appendMain(noticeStyle.Render("theme: " + cur + "   available: " + strings.Join(themeNames(), ", ")))
		return nil
	}
	if _, ok := palettes[rest]; !ok {
		m.appendMain(dimStyle.Render("unknown theme: " + rest + "   (try: " + strings.Join(themeNames(), ", ") + ")"))
		return nil
	}
	m.themeName = rest
	applyTheme(rest)
	m.appendMain(noticeStyle.Render("theme set to " + rest))
	return saveConfigCmd(m.config())
}

// handlePrompt views, sets, or clears the custom status prompt template.
func (m *model) handlePrompt(rest string) tea.Cmd {
	rest = strings.TrimSpace(rest)
	switch strings.ToLower(rest) {
	case "":
		if m.prompt == "" {
			m.appendMain(noticeStyle.Render("prompt: default   (set one, e.g. /prompt HP {hp}/{maxhp}  EN {ep}  {gold}g  {area})"))
		} else {
			m.appendMain(noticeStyle.Render("prompt: " + m.prompt + "   (/prompt off to restore default)"))
		}
		m.appendMain(dimStyle.Render("tokens: {hp} {maxhp} {ep} {maxep} {xp} {reqxp} {lvl} {gold} {area}"))
		return nil
	case "off", "default", "clear":
		m.prompt = ""
		m.appendMain(noticeStyle.Render("prompt reset to default"))
		return saveConfigCmd(m.config())
	}
	m.prompt = rest
	m.appendMain(noticeStyle.Render("prompt set"))
	return saveConfigCmd(m.config())
}

// handleAlarm views or sets the low HP/EN warning threshold.
func (m *model) handleAlarm(rest string) tea.Cmd {
	rest = strings.TrimSpace(strings.ToLower(rest))
	switch rest {
	case "":
		if m.alarmPct <= 0 {
			m.appendMain(noticeStyle.Render("low HP/EN alarm: off   (/alarm <pct> to enable)"))
		} else {
			m.appendMain(noticeStyle.Render(fmt.Sprintf("low HP/EN alarm: %d%%   (/alarm off to disable)", m.alarmPct)))
		}
		return nil
	case "off", "0":
		m.alarmPct = -1
		m.hpAlarmed, m.enAlarmed = false, false
		m.appendMain(noticeStyle.Render("alarm off"))
		return saveConfigCmd(m.config())
	}
	pct, err := strconv.Atoi(rest)
	if err != nil || pct < 1 || pct > 99 {
		m.appendMain(dimStyle.Render("usage: /alarm <1-99> | off"))
		return nil
	}
	m.alarmPct = pct
	m.hpAlarmed, m.enAlarmed = false, false
	m.appendMain(noticeStyle.Render(fmt.Sprintf("alarm set to %d%% — bell + warning when HP or EN drops that low", pct)))
	return saveConfigCmd(m.config())
}

func bellCmd() tea.Cmd {
	return func() tea.Msg {
		fmt.Fprint(os.Stderr, "\a")
		return nil
	}
}

func (m *model) reconnectNow() tea.Cmd {
	m.conn.Close()
	m.connected = false
	m.reconnectAttempt = 0
	m.conn = newConn(m.conn.transport, m.conn.addr)
	m.appendMain(dimStyle.Render("reconnecting…"))
	return connectCmd(m.conn)
}

// colorCode maps friendly color names to lipgloss/ANSI codes; unknown values
// pass through so raw codes ("196") and hex ("#ff8800") also work.
func colorCode(name string) string {
	switch strings.ToLower(name) {
	case "red":
		return "203"
	case "green":
		return "84"
	case "yellow":
		return "228"
	case "blue":
		return "39"
	case "cyan":
		return "117"
	case "magenta", "pink":
		return "213"
	case "orange":
		return "208"
	case "white":
		return "255"
	default:
		return name
	}
}
