package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// Fixed heights of the non-flexing rows in the vertical stack.
const (
	headerH = 1 // top status line
	statusH = 3 // HP/XP bar box (bordered)
	inputH  = 3 // command input box (bordered)
)

type focusTarget int

const (
	focusMain focusTarget = iota // OUTPUT
	focusChat                    // COMMS
	focusHere                    // left occupants column
	focusMap                     // MAP box
)

type model struct {
	conn     *conn
	autoCmds []string

	width, height int
	ready         bool

	// connection state
	connected bool
	connErr   error

	// panels
	main  viewport.Model
	here  viewport.Model // scrollable HERE (room occupants) list
	comms comms
	in    textinput.Model

	mainRaw string

	room   roomInfo
	status statusInfo
	target combatMsg // current attack target, for the enemy HP bar (Target=="" when idle)
	hasGot bool      // received at least one STATE frame

	alarmPct  int  // low HP/EN warn threshold (%); <=0 disables the alarm
	hpAlarmed bool // edge state so the alarm fires once per dip, not every tick
	enAlarmed bool

	themeName string // selected color theme name ("" = default)
	prompt    string // status prompt template ("" = default segments)

	focus      focusTarget
	showRight  bool
	fullMap    bool // full-screen map overlay (/map)
	layoutMode bool // interactive panel-resize mode (plain arrows resize)
	keyDebug   bool // echo key names to OUTPUT for diagnostics (/keys)
	mouseOn    bool // mouse capture enabled (/mouse toggles)
	compact    bool // no blank line between messages (/spacing toggles)

	// RULES overlay: view/edit aliases, highlights, triggers
	rulesOpen   bool
	rulesCursor int

	// macro editor overlay (nil when closed)
	macroEd *macroEditor

	mapper *mapper
	roster map[string]bool // confirmed online player names, for HERE highlighting

	// user configuration
	aliases    map[string]string
	highlights []rule
	triggers   []rule
	macros     map[string]macro // hotkey macros, keyed by slot ("f1".."f12")

	// session log + connection lifecycle
	logFile          *os.File
	quitting         bool // user asked to leave; suppress auto-reconnect
	reconnectAttempt int
	identity         string // current server-assigned login id (for auto-resume)

	// command history
	history []string
	histPos int
	draft   string // in-progress input preserved while browsing history

	// layout preferences, adjusted by the resize keys
	wantLeft  int // desired width of the left (HERE) column
	wantRight int // desired total width of the right column (MAP + COMMS)
	wantMapH  int // desired height of the MAP box within the right column

	// geometry computed by recalc and read by the view
	colLeft int
	colMain int
	rightW  int
	mapH    int
	midH    int
	macroH  int // height of the macro legend strip (0 or 1)
}

func newModel(c *conn, autoCmds []string) model {
	ti := textinput.New()
	ti.Prompt = "» "
	ti.Placeholder = "type a command (look, n, say hi, who, help)…"
	ti.CharLimit = 512
	ti.Focus()

	return model{
		conn:      c,
		autoCmds:  autoCmds,
		main:      viewport.New(0, 0),
		here:      viewport.New(0, 0),
		comms:     newComms(),
		in:        ti,
		showRight: true,
		focus:     focusMain,
		mouseOn:   true, // launched with tea.WithMouseCellMotion
		mapper:    newMapper(),
		roster:    map[string]bool{},
		aliases:   map[string]string{},
		macros:    map[string]macro{},
		wantLeft:  26,
		wantRight: 44,
		wantMapH:  13,
	}
}

// applyConfig overlays persisted configuration onto the model.
func (m *model) applyConfig(c clientConfig) {
	if c.Layout.LeftWidth > 0 {
		m.wantLeft = c.Layout.LeftWidth
	}
	m.wantRight = c.Layout.RightWidth
	m.wantMapH = c.Layout.MapHeight
	m.showRight = c.Layout.ShowRight
	if !m.showRight {
		m.focus = focusMain
	}
	m.compact = c.Compact
	m.comms.roomy = !c.Compact
	m.alarmPct = c.Alarm
	if m.alarmPct == 0 {
		m.alarmPct = 20 // default threshold; -1 disables
	}
	m.themeName = c.Theme
	m.prompt = c.Prompt
	applyTheme(c.Theme)
	if c.Aliases != nil {
		m.aliases = c.Aliases
	}
	if c.Macros != nil {
		m.macros = c.Macros
	}
	m.highlights = compileRules(c.Highlights)
	m.triggers = compileRules(c.Triggers)
}

// config snapshots the current state for persistence.
func (m model) config() clientConfig {
	return clientConfig{
		Layout:     layoutPrefs{LeftWidth: m.wantLeft, RightWidth: m.wantRight, MapHeight: m.wantMapH, ShowRight: m.showRight},
		Compact:    m.compact,
		Aliases:    m.aliases,
		Macros:     m.macros,
		Highlights: rulesToConfig(m.highlights),
		Triggers:   rulesToConfig(m.triggers),
		Alarm:      m.alarmPct,
		Theme:      m.themeName,
		Prompt:     m.prompt,
	}
}

// saveConfigCmd persists the config off the UI goroutine.
func saveConfigCmd(c clientConfig) tea.Cmd {
	return func() tea.Msg {
		_ = saveConfig(c)
		return nil
	}
}

// saveMapCmd persists the current map for this server off the UI goroutine.
func (m *model) saveMapCmd() tea.Cmd {
	d := m.mapper.snapshot()
	path := mapFilePath(m.conn.addr)
	return func() tea.Msg {
		_ = saveMapData(path, d)
		return nil
	}
}

// writeLog appends a line (ANSI stripped) to the session log when enabled.
func (m *model) writeLog(s string) {
	if m.logFile == nil {
		return
	}
	fmt.Fprintln(m.logFile, ansi.Strip(s))
}

// backoff returns the reconnect delay for a given attempt (1s,2s,…, capped 30s).
func backoff(attempt int) time.Duration {
	if attempt > 6 {
		attempt = 6
	}
	d := time.Second << (attempt - 1)
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}

func (m model) Init() tea.Cmd {
	return tea.Batch(connectCmd(m.conn), textinput.Blink)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ready = true
		m.recalc()
		return m, nil

	case connectedMsg:
		m.connected = true
		m.connErr = nil
		m.reconnectAttempt = 0
		m.appendMain(dimStyle.Render(fmt.Sprintf("✓ connected to %s (%s)", m.conn.addr, m.conn.transport)))
		for _, c := range m.autoCmds {
			cmds = append(cmds, sendCmd(m.conn, c))
		}
		cmds = append(cmds, waitForEvent(m.conn))
		return m, tea.Batch(cmds...)

	case disconnectedMsg:
		m.connected = false
		m.connErr = msg.err
		if msg.err != nil {
			m.appendMain(badStyle.Render("✗ disconnected: " + msg.err.Error()))
		} else {
			m.appendMain(badStyle.Render("✗ disconnected"))
		}
		if m.quitting {
			return m, tea.Quit
		}
		m.reconnectAttempt++
		d := backoff(m.reconnectAttempt)
		m.appendMain(dimStyle.Render(fmt.Sprintf("reconnecting in %s…", d.Round(time.Second))))
		return m, tea.Tick(d, func(time.Time) tea.Msg { return reconnectMsg{} })

	case reconnectMsg:
		m.conn = newConn(m.conn.transport, m.conn.addr)
		return m, connectCmd(m.conn)

	case chunkMsg:
		cmd := m.handleChunk(msg.raw)
		return m, tea.Batch(cmd, waitForEvent(m.conn))

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward anything else (cursor blink, etc.) to the input.
	var cmd tea.Cmd
	m.in, cmd = m.in.Update(msg)
	return m, cmd
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.keyDebug {
		m.appendMain(dimStyle.Render("key: " + msg.String()))
	}
	if m.macroEd != nil {
		return m.handleMacroEditorKey(msg)
	}
	if m.rulesOpen {
		return m.handleRulesKey(msg)
	}
	if m.layoutMode {
		return m.handleLayoutKey(msg)
	}

	// Function keys fire macros (only real F-keys, so typing digits is unaffected).
	// An unbound slot is a no-op.
	if slot := fKeySlot(msg.String()); slot != "" {
		return m, m.fireMacro(slot)
	}

	// Shifted number-row keys (!@#$%…) fire macros 1–10, but only when the input
	// line is empty and the slot is bound — so those symbols can still be typed
	// into messages, and F-keys remain the mid-typing way to fire a macro.
	if m.in.Value() == "" {
		if slot := macroSymbolSlot(msg); slot != "" {
			if _, ok := m.macros[slot]; ok {
				return m, m.fireMacro(slot)
			}
		}
	}

	switch msg.String() {
	case "ctrl+c":
		// Persist synchronously so it lands before we exit.
		_ = saveConfig(m.config())
		_ = saveMapData(mapFilePath(m.conn.addr), m.mapper.snapshot())
		m.conn.Close()
		return m, tea.Quit

	case "ctrl+o":
		m.layoutMode = true
		if m.focus == focusMain { // OUTPUT is the flex filler; pick a resizable pane
			m.cycleFocus(1)
		}
		return m, nil

	// Movement: macOS terminals send Option+Left/Right as alt+b / alt+f
	// (word-jump), so bind those too. Option+Up/Down already arrive as alt+up/down.
	case "alt+b":
		return m, m.dispatch("west")
	case "alt+f":
		return m, m.dispatch("east")

	case "enter":
		line := m.in.Value()
		m.in.SetValue("")
		return m, m.submit(line)

	case "tab":
		// Empty input: cycle pane focus. Otherwise: complete the last token.
		if strings.TrimSpace(m.in.Value()) == "" {
			m.cycleFocus(1)
			return m, nil
		}
		m.completeInput()
		return m, nil

	case "shift+tab":
		m.cycleFocus(-1)
		return m, nil

	case "alt+up":
		return m, m.dispatch("north")
	case "alt+down":
		return m, m.dispatch("south")
	case "alt+left":
		return m, m.dispatch("west")
	case "alt+right":
		return m, m.dispatch("east")

	case "ctrl+g":
		m.fullMap = !m.fullMap
		return m, nil

	case "ctrl+a":
		m.rulesOpen = true
		m.rulesCursor = 0
		m.fullMap = false
		return m, nil

	case "esc":
		if m.fullMap {
			m.fullMap = false
		}
		return m, nil

	case "ctrl+n":
		m.comms.next()
		return m, nil
	case "ctrl+p":
		m.comms.prev()
		return m, nil

	case "ctrl+t":
		m.showRight = !m.showRight
		if !m.showRight {
			m.focus = focusMain
		}
		m.recalc()
		return m, saveConfigCmd(m.config())

	// Resize the focused pane (HERE/MAP/COMMS; OUTPUT is the flex filler).
	case "ctrl+right":
		m.resizeFocused(4, 0)
		return m, saveConfigCmd(m.config())
	case "ctrl+left":
		m.resizeFocused(-4, 0)
		return m, saveConfigCmd(m.config())
	case "ctrl+up":
		m.resizeFocused(0, 2)
		return m, saveConfigCmd(m.config())
	case "ctrl+down":
		m.resizeFocused(0, -2)
		return m, saveConfigCmd(m.config())

	case "ctrl+l":
		m.mainRaw = ""
		m.main.SetContent("")
		return m, nil

	case "pgup", "pgdown", "ctrl+u", "ctrl+d":
		if vp := m.activeViewport(); vp != nil {
			var cmd tea.Cmd
			*vp, cmd = vp.Update(msg)
			return m, cmd
		}
		return m, nil

	case "up":
		if m.histPos > 0 {
			if m.histPos == len(m.history) {
				m.draft = m.in.Value() // stash the in-progress line
			}
			m.histPos--
			m.in.SetValue(m.history[m.histPos])
			m.in.CursorEnd()
		}
		return m, nil

	case "down":
		if m.histPos < len(m.history) {
			m.histPos++
			if m.histPos == len(m.history) {
				m.in.SetValue(m.draft) // restore the in-progress line
			} else {
				m.in.SetValue(m.history[m.histPos])
			}
			m.in.CursorEnd()
		}
		return m, nil
	}

	var cmd tea.Cmd
	m.in, cmd = m.in.Update(msg)
	return m, cmd
}

// handleLayoutKey drives the interactive resize mode using plain arrow keys,
// which every terminal delivers reliably (unlike ctrl/alt+arrow).
func (m model) handleLayoutKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		_ = saveConfig(m.config())
		_ = saveMapData(mapFilePath(m.conn.addr), m.mapper.snapshot())
		m.conn.Close()
		return m, tea.Quit
	case "tab": // pick which pane to resize
		m.cycleFocus(1)
		return m, nil
	case "shift+tab":
		m.cycleFocus(-1)
		return m, nil
	case "right":
		m.resizeFocused(4, 0)
		return m, saveConfigCmd(m.config())
	case "left":
		m.resizeFocused(-4, 0)
		return m, saveConfigCmd(m.config())
	case "up":
		m.resizeFocused(0, 2)
		return m, saveConfigCmd(m.config())
	case "down":
		m.resizeFocused(0, -2)
		return m, saveConfigCmd(m.config())
	case "esc", "enter", "ctrl+o", "q":
		m.layoutMode = false
		return m, nil
	}
	return m, nil
}

// handleChunk classifies one server message, updates the relevant panels, and
// returns commands for any fired triggers or map persistence.
func (m *model) handleChunk(raw string) tea.Cmd {
	m.updateRoster(raw)
	r := classify(raw)
	var cmds []tea.Cmd

	if r.status != nil {
		m.status = *r.status
		m.hasGot = true
		cmds = append(cmds, m.checkAlarms()...)
	}
	if r.room != nil {
		if r.room.Title != m.room.Title {
			m.target = combatMsg{} // entered a new room; drop the enemy bar
		}
		m.room = *r.room
		m.refreshHere()
		if m.mapper.arrive(r.room.Title, r.room.Exits) {
			cmds = append(cmds, m.saveMapCmd())
		}
	}
	if r.info != nil {
		// room.info is authoritative for name/exits (the text scrape is a
		// fallback for transports that don't get events). Occupants are left to
		// room.contents / the text scrape.
		if r.info.Title != m.room.Title {
			m.target = combatMsg{}
		}
		m.room.Title = r.info.Title
		m.room.Exits = r.info.Exits
		if m.mapper.arrive(r.info.Title, r.info.Exits) {
			cmds = append(cmds, m.saveMapCmd())
		}
	}
	if r.contents != nil {
		m.applyContents(*r.contents)
	}
	if r.comms != nil {
		tab, line := renderComms(r.comms)
		m.appendChatTo(tab, applyHighlights(line, m.highlights))
		cmds = append(cmds, m.fireTriggers(line)...)
	}
	if r.combat != nil {
		m.target = *r.combat
	}
	if len(r.macros) > 0 {
		applied := 0
		for _, b := range r.macros {
			if b.Cmd == "" || b.Slot < 1 || b.Slot > macroSlots {
				continue
			}
			m.macros[fmt.Sprintf("f%d", b.Slot)] = macro{Label: b.Label, Cmd: b.Cmd}
			applied++
		}
		if applied > 0 {
			m.recalc() // the bar may appear or grow
			m.appendMain(noticeStyle.Render(fmt.Sprintf("✓ %d class hotkeys loaded onto your bar", applied)))
			cmds = append(cmds, saveConfigCmd(m.config()))
		}
	}
	if r.identity != "" && r.identity != m.identity {
		fresh := m.identity == ""
		m.identity = r.identity
		cmds = append(cmds, saveIdentityCmd(m.conn.addr, r.identity))
		if fresh {
			m.appendMain(noticeStyle.Render("✓ character saved — it'll resume automatically next time you connect"))
		}
	}
	if r.toChat != "" {
		m.appendChat(applyHighlights(r.toChat, m.highlights))
		cmds = append(cmds, m.fireTriggers(r.toChat)...)
	}
	if r.toMain != "" {
		m.appendMain(applyHighlights(r.toMain, m.highlights))
		cmds = append(cmds, m.fireTriggers(r.toMain)...)
	}
	return tea.Batch(cmds...)
}

// applyContents updates the live HERE list from a room.contents event, marking
// present players in the roster so they're highlighted.
func (m *model) applyContents(c roomContents) {
	occ := make([]string, 0, len(c.Players)+len(c.NPCs)+len(c.Corpses)+len(c.Items))
	for _, p := range c.Players {
		m.roster[p] = true
		occ = append(occ, p)
	}
	occ = append(occ, c.NPCs...)
	occ = append(occ, c.Corpses...)
	occ = append(occ, c.Items...)
	m.room.Occupants = occ
	m.refreshHere()
}

// refreshHere re-renders the scrollable HERE list, keeping the view pinned to
// the bottom when it was already there so new arrivals stay visible.
func (m *model) refreshHere() {
	atBottom := m.here.AtBottom()
	m.here.SetContent(hereList(m.room.Occupants, m.roster))
	if atBottom {
		m.here.GotoBottom()
	}
}

// updateRoster tracks confirmed online players from STATUS join/leave notices
// and any who-table output, so HERE can highlight which occupants are players.
func (m *model) updateRoster(raw string) {
	raw = strings.TrimSpace(raw)
	if mm := joinRe.FindStringSubmatch(raw); mm != nil {
		m.roster[mm[1]] = true
		return
	}
	if mm := leaveRe.FindStringSubmatch(raw); mm != nil {
		delete(m.roster, mm[1])
		return
	}
	for _, name := range parseWhoNames(raw) {
		m.roster[name] = true
	}
}

// handleMouse routes wheel events to the panel under the cursor and left-clicks
// to focus a panel or switch a COMMS tab.
func (m model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown:
		if vp := m.viewportAt(msg.X, msg.Y); vp != nil {
			*vp, _ = vp.Update(msg)
		}
	case msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft:
		m.handleClick(msg.X, msg.Y)
	}
	return m, nil
}

// viewportAt returns the scrollable panel under a screen cell, or nil.
func (m *model) viewportAt(x, y int) *viewport.Model {
	middleTop := headerH + statusH
	if y < middleTop || y >= middleTop+m.midH {
		return nil // header / status / input rows
	}
	if m.showRight && m.rightW > 0 && x >= m.colLeft+m.colMain {
		if y >= middleTop+m.mapH { // below the MAP box = COMMS
			return &m.comms.vp
		}
		return nil // MAP box is not scrollable
	}
	if x >= m.colLeft && x < m.colLeft+m.colMain {
		return &m.main
	}
	if x < m.colLeft {
		return &m.here
	}
	return nil
}

// handleClick focuses the clicked panel, or switches COMMS tabs when the tab
// bar is clicked.
func (m *model) handleClick(x, y int) {
	middleTop := headerH + statusH
	if y < middleTop || y >= middleTop+m.midH {
		return
	}
	rightStart := m.colLeft + m.colMain
	switch {
	case x < m.colLeft:
		m.focus = focusHere
	case x < rightStart:
		m.focus = focusMain
	case m.showRight && m.rightW > 0:
		if m.mapH > 0 && y < middleTop+m.mapH {
			m.focus = focusMap
		} else {
			m.focus = focusChat
			if y == middleTop+m.mapH+1 { // COMMS tab bar row
				if idx := m.comms.tabAt(x - (rightStart + 1)); idx >= 0 {
					m.comms.switchTo(idx)
				}
			}
		}
	}
}

// activeViewport returns the focused scrollable pane, or nil for panes that
// don't scroll (HERE, MAP).
func (m *model) activeViewport() *viewport.Model {
	switch m.focus {
	case focusMain:
		return &m.main
	case focusHere:
		return &m.here
	case focusChat:
		if m.showRight && m.rightW > 0 {
			return &m.comms.vp
		}
	}
	return nil
}

// focusables lists the panes Tab can cycle through, in display order.
func (m model) focusables() []focusTarget {
	fs := []focusTarget{focusHere, focusMain}
	if m.showRight && m.rightW > 0 {
		if m.mapH > 0 {
			fs = append(fs, focusMap)
		}
		fs = append(fs, focusChat)
	}
	return fs
}

// cycleFocus moves focus to the next (dir +1) or previous (dir -1) visible pane.
func (m *model) cycleFocus(dir int) {
	fs := m.focusables()
	cur := 0
	for i, f := range fs {
		if f == m.focus {
			cur = i
		}
	}
	m.focus = fs[((cur+dir)%len(fs)+len(fs))%len(fs)]
}

// resizeFocused grows/shrinks the focused pane: dx columns of width, dy rows of
// height (positive = taller). OUTPUT is the flex filler and isn't resized.
func (m *model) resizeFocused(dx, dy int) {
	switch m.focus {
	case focusHere:
		m.wantLeft += dx
	case focusMap:
		m.wantRight += dx
		m.wantMapH += dy
	case focusChat:
		m.wantRight += dx
		m.wantMapH -= dy // growing COMMS shrinks the MAP above it
	}
	m.recalc()
}

// checkAlarms rings the bell and warns when HP or EN crosses below the alarm
// threshold. Edge-triggered (hpAlarmed/enAlarmed) so it fires once per dip, then
// rearms when the bar recovers.
func (m *model) checkAlarms() []tea.Cmd {
	if m.alarmPct <= 0 {
		return nil
	}
	var cmds []tea.Cmd
	if m.status.HasHP && m.status.MaxHP > 0 && m.status.HP > 0 {
		if m.status.HP*100/m.status.MaxHP <= m.alarmPct {
			if !m.hpAlarmed {
				m.hpAlarmed = true
				m.appendMain(alarmStyle.Render(fmt.Sprintf("⚠ LOW HEALTH  %d/%d", m.status.HP, m.status.MaxHP)))
				cmds = append(cmds, bellCmd())
			}
		} else {
			m.hpAlarmed = false
		}
	}
	if m.status.HasEP && m.status.MaxEP > 0 {
		if m.status.EP*100/m.status.MaxEP <= m.alarmPct {
			if !m.enAlarmed {
				m.enAlarmed = true
				m.appendMain(alarmStyle.Render(fmt.Sprintf("⚠ LOW ENDURANCE  %d/%d", m.status.EP, m.status.MaxEP)))
				cmds = append(cmds, bellCmd())
			}
		} else {
			m.enAlarmed = false
		}
	}
	return cmds
}

func (m *model) appendMain(s string) {
	m.writeLog(s)
	m.mainRaw = joinMessage(m.mainRaw, s, !m.compact)
	atBottom := m.main.AtBottom()
	m.main.SetContent(wrap(m.mainRaw, m.main.Width))
	if atBottom {
		m.main.GotoBottom()
	}
}

func (m *model) appendChat(s string) {
	m.appendChatTo(routeComms(ansi.Strip(s)), s)
}

// appendChatTo appends a chat line to a named COMMS tab. Used directly for
// structured comms events (which carry their own channel) and via appendChat
// for text whose tab is inferred by regex.
func (m *model) appendChatTo(tab, s string) {
	line := colorizeChat(s)
	m.writeLog(line)
	m.comms.append(tab, line)
}

// joinMessage appends a message to a transcript buffer. When roomy, messages
// are separated by a blank line so distinct incoming events don't run together.
func joinMessage(buf, s string, roomy bool) string {
	if buf == "" {
		return s
	}
	if roomy {
		return buf + "\n\n" + s
	}
	return buf + "\n" + s
}

// recalc recomputes panel geometry from the current terminal size.
func (m *model) recalc() {
	if !m.ready {
		return
	}

	// The macro legend strip takes one row above the input, but only when macros
	// exist — otherwise it costs nothing.
	m.macroH = 0
	if len(m.macros) > 0 {
		m.macroH = 1
	}

	m.midH = max(3, m.height-headerH-statusH-inputH-m.macroH)

	m.colLeft = clamp(m.wantLeft, 14, max(14, m.width/3))

	// Right column width: clamp the preference so MAIN keeps at least 20 cols.
	m.rightW = 0
	if m.showRight {
		m.rightW = clamp(m.wantRight, 24, max(24, m.width-m.colLeft-20))
	}

	m.colMain = m.width - m.colLeft - m.rightW
	if m.colMain < 16 {
		// Keep MAIN usable by stealing from the right column first, then left.
		deficit := 16 - m.colMain
		take := min(deficit, m.rightW)
		m.rightW -= take
		deficit -= take
		if deficit > 0 {
			m.colLeft = max(8, m.colLeft-deficit)
		}
		m.colMain = m.width - m.colLeft - m.rightW
	}

	// MAP/COMMS split inside the right column. A map shorter than 4 rows is
	// hidden so COMMS gets the whole column.
	m.mapH = 0
	if m.rightW > 0 {
		m.mapH = clamp(m.wantMapH, 0, max(0, m.midH-6))
		if m.mapH < 6 { // needs room for the 2-line header + a little graph
			m.mapH = 0
		}
	}

	// Panels reserve one inner line for their title plus two for the border,
	// so a scrollable viewport gets (box height - 3) rows.
	m.main.Width = max(1, m.colMain-2)
	m.main.Height = max(1, m.midH-3)
	m.here.Width = max(1, m.colLeft-2)
	m.here.Height = max(1, m.midH-3)
	m.refreshHere()
	if m.rightW > 0 {
		// COMMS box reserves one inner line for the tab bar plus two for the border.
		commsH := m.midH - m.mapH
		m.comms.setSize(max(1, m.rightW-2), max(1, commsH-3))
	}
	m.in.Width = max(8, m.width-6)

	// Re-wrap the main transcript to the new width.
	m.main.SetContent(wrap(m.mainRaw, m.main.Width))
	m.main.GotoBottom()

	// If the focused pane is no longer visible, fall back to OUTPUT.
	visible := false
	for _, f := range m.focusables() {
		if f == m.focus {
			visible = true
		}
	}
	if !visible {
		m.focus = focusMain
	}

	m.sizeMacroEditor()
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
