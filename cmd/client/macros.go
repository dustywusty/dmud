package main

import (
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// macroSlots is the number of bindable hotkeys: F1–F12.
const macroSlots = 12

// macro is a command (or ;-separated sequence) bound to a function key. Label is
// an optional short name shown on the macro bar; when empty the command itself is
// shown. The command flows through the normal input pipeline, so it inherits
// aliases, speedwalk, and ;-sequences.
type macro struct {
	Label string `json:"label,omitempty"`
	Cmd   string `json:"cmd"`
}

// display is the short text shown for a macro: its label if set, else a
// single-line preview of its command (multi-line bodies are summarized).
func (mac macro) display() string {
	if mac.Label != "" {
		return mac.Label
	}
	return macroOneLine(mac.Cmd)
}

// macroOneLine collapses a (possibly multi-line) command to a single line for
// the bar / overlay rows: the first line, with an ellipsis if there are more.
func macroOneLine(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if i := strings.IndexByte(cmd, '\n'); i >= 0 {
		return strings.TrimSpace(cmd[:i]) + " …"
	}
	return cmd
}

// macroRowText is the styled description of a macro in the RULES overlay: its
// label (if any) plus a dim one-line preview of the command.
func macroRowText(mac macro) string {
	if mac.Label != "" {
		return chatNameStyle.Render(mac.Label) + dimStyle.Render("  "+macroOneLine(mac.Cmd))
	}
	return macroOneLine(mac.Cmd)
}

// macroSlot normalizes a user-typed key ("3", "f3", "F3") to a canonical slot
// name ("f3"), or "" if it isn't a valid F1–F12 slot. Used to parse /macro args.
func macroSlot(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.TrimPrefix(s, "f")
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 || n > macroSlots {
		return ""
	}
	return "f" + strconv.Itoa(n)
}

// fKeySlot returns the slot a pressed key maps to, but only for genuine function
// keys ("f1".."f12") — never bare digits, so typing "1" into the input is safe.
// F-keys are the mid-typing fallback for the Shift+digit bindings below.
func fKeySlot(key string) string {
	if len(key) < 2 || key[0] != 'f' {
		return ""
	}
	n, err := strconv.Atoi(key[1:])
	if err != nil || n < 1 || n > macroSlots {
		return ""
	}
	return key
}

// shiftDigits maps the shifted number-row symbols (US keyboard) to macro slots
// 1–10. Terminals deliver Shift+1 as "!", Shift+2 as "@", and so on — there is
// no separate "shift+1" key event — so these symbols are how Shift+<digit> is
// recognized.
var shiftDigits = map[rune]int{
	'!': 1, '@': 2, '#': 3, '$': 4, '%': 5,
	'^': 6, '&': 7, '*': 8, '(': 9, ')': 10,
}

// macroSymbolSlot returns the slot a shifted-digit keypress maps to ("!" → "f1"),
// or "" for anything else. Alt-modified and multi-rune presses are ignored.
func macroSymbolSlot(msg tea.KeyMsg) string {
	if msg.Type != tea.KeyRunes || msg.Alt || len(msg.Runes) != 1 {
		return ""
	}
	if n, ok := shiftDigits[msg.Runes[0]]; ok {
		return "f" + strconv.Itoa(n)
	}
	return ""
}

// slotKeyLabel is how a slot is shown to the user: "⇧1".."⇧0" for the Shift+digit
// slots 1–10, or the function-key name ("F11", "F12") for the rest.
func slotKeyLabel(slot string) string {
	if n := macroSlotNum(slot); n >= 1 && n <= 10 {
		return "⇧" + strconv.Itoa(n%10)
	}
	return strings.ToUpper(slot)
}

// macroSlotNum returns the 1-based number for a slot name ("f3" → 3), or 0.
func macroSlotNum(slot string) int {
	n, _ := strconv.Atoi(strings.TrimPrefix(slot, "f"))
	return n
}

// macroOrder returns the configured slots sorted F1…F12.
func (m model) macroOrder() []string {
	slots := make([]string, 0, len(m.macros))
	for s := range m.macros {
		slots = append(slots, s)
	}
	sort.Slice(slots, func(i, j int) bool { return macroSlotNum(slots[i]) < macroSlotNum(slots[j]) })
	return slots
}

// fireMacro runs the macro bound to a slot ("f1"…), echoing it like a typed
// command and expanding aliases/speedwalk/; exactly as submit does. An unbound
// slot is a silent no-op, so dead F-keys do nothing.
func (m *model) fireMacro(slot string) tea.Cmd {
	mac, ok := m.macros[slot]
	if !ok {
		return nil
	}
	var cmds []tea.Cmd
	// Each line of the body is run in order; a line may itself hold ;-separated
	// commands, aliases, and speedwalk (the same pipeline as typed input).
	for _, line := range strings.Split(mac.Cmd, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m.appendMain(dimStyle.Render("» ") + line)
		for _, c := range m.expand(line) {
			if c == "exit" {
				m.quitting = true
			}
			cmds = append(cmds, m.dispatch(c))
		}
	}
	return tea.Batch(cmds...)
}

// handleMacro implements /macro: list, set (with optional "label = cmd"), or
// clear a hotkey binding.
func (m *model) handleMacro(rest string) tea.Cmd {
	if rest == "" {
		m.appendMain(m.macroList())
		return nil
	}

	slotRaw, body, _ := strings.Cut(rest, " ")

	// /macro edit <slot> opens the multi-line editor (create or change).
	if strings.EqualFold(slotRaw, "edit") {
		target := macroSlot(strings.TrimSpace(body))
		if target == "" {
			m.appendMain(badStyle.Render("usage: /macro edit <1-12>   (opens the multi-line editor)"))
			return nil
		}
		return m.openMacroEditor(target)
	}

	slot := macroSlot(slotRaw)
	if slot == "" {
		m.appendMain(badStyle.Render("usage: /macro <1-12> [label =] <command>, or /macro edit <1-12>"))
		return nil
	}
	body = strings.TrimSpace(body)

	if body == "" { // clear
		if _, ok := m.macros[slot]; ok {
			delete(m.macros, slot)
			m.appendMain(noticeStyle.Render("macro cleared: " + slotKeyLabel(slot)))
		} else {
			m.appendMain(dimStyle.Render(slotKeyLabel(slot) + " is not bound"))
		}
		m.recalc() // the bar may appear or disappear
		return saveConfigCmd(m.config())
	}

	mac := macro{Cmd: body}
	if label, cmd, ok := strings.Cut(body, " = "); ok {
		mac.Label = strings.TrimSpace(label)
		mac.Cmd = strings.TrimSpace(cmd)
	}
	if mac.Cmd == "" {
		m.appendMain(badStyle.Render("a macro needs a command"))
		return nil
	}
	if m.macros == nil {
		m.macros = map[string]macro{}
	}
	m.macros[slot] = mac
	m.appendMain(noticeStyle.Render("macro set: " + slotKeyLabel(slot) + " → " + macroOneLine(mac.Cmd)))
	m.recalc()
	return saveConfigCmd(m.config())
}

// macroList renders the configured macros for the OUTPUT panel.
func (m model) macroList() string {
	order := m.macroOrder()
	if len(order) == 0 {
		return dimStyle.Render("no macros set — try /macro 1 look")
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render("macros"))
	for _, slot := range order {
		mac := m.macros[slot]
		label := ""
		if mac.Label != "" {
			label = chatNameStyle.Render(mac.Label) + dimStyle.Render(" = ")
		}
		b.WriteString("\n  " + macroKeyStyle.Render(slotKeyLabel(slot)) + "  " + label + mac.Cmd)
	}
	return b.String()
}

// renderMacroBar draws the one-line F-key legend above the input, or "" when no
// macros are set (so it costs no vertical space).
func (m model) renderMacroBar() string {
	order := m.macroOrder()
	if len(order) == 0 {
		return ""
	}
	parts := make([]string, 0, len(order))
	for _, slot := range order {
		parts = append(parts, macroKeyStyle.Render(slotKeyLabel(slot))+" "+m.macros[slot].display())
	}
	line := " " + strings.Join(parts, dimStyle.Render("  ·  "))
	return lipgloss.NewStyle().MaxWidth(m.width).Render(line)
}
