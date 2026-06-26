package main

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// commsTab is one channel in the COMMS panel: a named, scrollable buffer.
type commsTab struct {
	name   string
	raw    string
	unread bool
}

// comms is the tabbed communication panel. "Local" (say) and "Shout" always
// exist; a tab per person is created on demand for private messages.
type comms struct {
	tabs   []*commsTab
	active int
	vp     viewport.Model
	roomy  bool // blank line between messages
}

func newComms() comms {
	return comms{
		tabs:  []*commsTab{{name: "Local"}, {name: "Shout"}},
		vp:    viewport.New(0, 0),
		roomy: true,
	}
}

func (c *comms) find(name string) *commsTab {
	for _, t := range c.tabs {
		if strings.EqualFold(t.name, name) {
			return t
		}
	}
	t := &commsTab{name: name}
	c.tabs = append(c.tabs, t)
	return t
}

// append adds a line to the named tab, marking it unread unless it is active.
func (c *comms) append(name, line string) {
	t := c.find(name)
	t.raw = joinMessage(t.raw, line, c.roomy)
	if c.activeTab() == t {
		c.refresh()
	} else {
		t.unread = true
	}
}

func (c *comms) activeTab() *commsTab {
	if c.active < 0 || c.active >= len(c.tabs) {
		return nil
	}
	return c.tabs[c.active]
}

func (c *comms) refresh() {
	t := c.activeTab()
	if t == nil {
		return
	}
	atBottom := c.vp.AtBottom()
	c.vp.SetContent(wrap(t.raw, c.vp.Width))
	if atBottom {
		c.vp.GotoBottom()
	}
}

func (c *comms) setSize(w, h int) {
	c.vp.Width = w
	c.vp.Height = h
	c.refresh()
}

func (c *comms) next() { c.switchTo(c.active + 1) }
func (c *comms) prev() { c.switchTo(c.active - 1) }

func (c *comms) switchTo(i int) {
	n := len(c.tabs)
	if n == 0 {
		return
	}
	c.active = ((i % n) + n) % n
	if t := c.activeTab(); t != nil {
		t.unread = false
		c.vp.SetContent(wrap(t.raw, c.vp.Width))
		c.vp.GotoBottom()
	}
}

// tabAt maps a column offset within the tab bar to a tab index, or -1 (e.g. a
// separator). The width math must match tabBar's rendering.
func (c *comms) tabAt(offset int) int {
	if offset < 0 {
		return -1
	}
	x := 0
	for i, t := range c.tabs {
		w := lipgloss.Width(t.name)
		if t.unread && i != c.active {
			w++ // the unread "•"
		}
		if offset >= x && offset < x+w {
			return i
		}
		x += w + lipgloss.Width(" · ") // separator
	}
	return -1
}

// tabBar renders the channel labels: active is highlighted, unread channels get
// a dot, the rest are dim.
func (c *comms) tabBar() string {
	parts := make([]string, len(c.tabs))
	for i, t := range c.tabs {
		switch {
		case i == c.active:
			parts[i] = commsActiveStyle.Render(t.name)
		case t.unread:
			parts[i] = commsUnreadStyle.Render(t.name + "•")
		default:
			parts[i] = dimStyle.Render(t.name)
		}
	}
	return strings.Join(parts, dimStyle.Render(" · "))
}

// --- routing ---

var (
	reShout      = regexp.MustCompile(`(?i)(^you shout\b|\bshouts:)`)
	reTellOut    = regexp.MustCompile(`(?i)^you tell (\S+?):`)
	reTellIn     = regexp.MustCompile(`(?i)^(\S+) tells you\b`)
	reWhisperOut = regexp.MustCompile(`(?i)^you whisper(?: to)? (\S+?):`)
	reWhisperIn  = regexp.MustCompile(`(?i)^(\S+) whispers\b`)
)

// renderComms turns a structured comms event into its display tab and line. It
// mirrors the server's legacy plain text so tag and non-tag clients read the
// same, but routes by the event's channel rather than scraping the text.
func renderComms(ev *commsMsg) (tab, line string) {
	switch ev.Channel {
	case "tell":
		if ev.Self {
			return ev.To, "You tell " + ev.To + ": " + ev.Text
		}
		return ev.From, ev.From + " tells you: " + ev.Text
	case "shout":
		if ev.Self {
			return "Shout", "You shout: " + ev.Text
		}
		return "Shout", ev.From + " shouts: " + ev.Text
	default: // say and anything else
		if ev.Self {
			return "Local", "You say: " + ev.Text
		}
		return "Local", ev.From + " says: " + ev.Text
	}
}

// routeComms picks the tab for a chat line: a person's name for private
// messages, "Shout" for shouts, otherwise "Local" (say and anything else).
func routeComms(text string) string {
	text = strings.TrimSpace(text)
	for _, re := range []*regexp.Regexp{reTellOut, reTellIn, reWhisperOut, reWhisperIn} {
		if m := re.FindStringSubmatch(text); m != nil {
			return m[1]
		}
	}
	if reShout.MatchString(text) {
		return "Shout"
	}
	return "Local"
}

// clipWidth truncates a (possibly styled) line to w cells.
func clipWidth(s string, w int) string {
	return lipgloss.NewStyle().MaxWidth(w).Render(s)
}
