package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRouteComms(t *testing.T) {
	cases := map[string]string{
		"Gandalf says: hi":      "Local",
		"You say: hi":           "Local",
		"Frodo shouts: help":    "Shout",
		"You shout: hey":        "Shout",
		"Alice tells you: psst": "Alice",
		"You tell Bob: hello":   "Bob",
		"Sam whispers: quietly": "Sam",
		"unmatched chatter":     "Local", // default channel
	}
	for line, want := range cases {
		if got := routeComms(line); got != want {
			t.Errorf("routeComms(%q) = %q, want %q", line, got, want)
		}
	}
}

func TestCommsTabsAndUnread(t *testing.T) {
	c := newComms()
	c.setSize(40, 6)

	// Local is active (index 0); a shout marks the Shout tab unread.
	c.append("Shout", "Frodo shouts: help")
	if !c.find("Shout").unread {
		t.Error("an inactive tab should be marked unread")
	}

	// A private message auto-creates a per-person tab.
	c.append("Alice", "Alice tells you: hi")
	if len(c.tabs) != 3 {
		t.Errorf("private message should create a tab; have %d tabs", len(c.tabs))
	}

	// Switching to Shout clears its unread and shows its content.
	for c.activeTab().name != "Shout" {
		c.next()
	}
	if c.find("Shout").unread {
		t.Error("switching to a tab should clear its unread flag")
	}
	if view := ansi.Strip(c.vp.View()); !strings.Contains(view, "Frodo shouts") {
		t.Errorf("active tab should show its own content; got %q", view)
	}

	// Appending to the active tab updates the viewport live.
	c.append("Shout", "Sam shouts: over here")
	if view := ansi.Strip(c.vp.View()); !strings.Contains(view, "over here") {
		t.Errorf("active tab should update live; got %q", view)
	}
}
