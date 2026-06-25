package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestMapperLinksRoomsByMovement(t *testing.T) {
	mp := newMapper()
	mp.arrive("Crossroads", []string{"north"}) // first room anchors at origin
	mp.noteCommand("north")
	mp.arrive("Forest Edge", []string{"south"})

	if mp.cur != "Forest Edge" {
		t.Fatalf("current = %q, want Forest Edge", mp.cur)
	}
	cross, forest := mp.rooms["Crossroads"], mp.rooms["Forest Edge"]
	if cross == nil || forest == nil {
		t.Fatal("both rooms should be mapped")
	}
	if cross.pos != (coord{0, 0}) {
		t.Errorf("Crossroads at %+v, want origin", cross.pos)
	}
	if forest.pos != (coord{0, -1}) {
		t.Errorf("Forest Edge at %+v, want north of origin", forest.pos)
	}

	view := ansi.Strip(mp.render(13, 7))
	if !strings.Contains(view, "[@]") {
		t.Errorf("current room should render as a box [@]; got:\n%s", view)
	}
	if strings.Count(view, "]") < 2 {
		t.Errorf("both rooms should render as boxes; got:\n%s", view)
	}
	if !strings.Contains(view, "│") {
		t.Errorf("the reciprocal link should draw a vertical connector; got:\n%s", view)
	}
}

func TestMapperReturnKeepsCoordinates(t *testing.T) {
	mp := newMapper()
	mp.arrive("A", []string{"north"})
	mp.noteCommand("n")
	mp.arrive("B", []string{"south"})
	mp.noteCommand("s")
	mp.arrive("A", []string{"north"}) // walk back

	if len(mp.rooms) != 2 {
		t.Errorf("expected 2 rooms, got %d", len(mp.rooms))
	}
	if mp.cur != "A" {
		t.Errorf("current = %q, want A", mp.cur)
	}
	if mp.rooms["A"].pos != (coord{0, 0}) || mp.rooms["B"].pos != (coord{0, -1}) {
		t.Errorf("coords drifted: A=%+v B=%+v", mp.rooms["A"].pos, mp.rooms["B"].pos)
	}
}

func TestMapperFailedMoveDoesNotCreateGhostRoom(t *testing.T) {
	mp := newMapper()
	mp.arrive("A", []string{"north"})
	mp.noteCommand("east") // there is no east exit; server re-describes A
	mp.arrive("A", []string{"north"})

	if len(mp.rooms) != 1 {
		t.Errorf("a failed move should not add a room; got %d rooms", len(mp.rooms))
	}
	if len(mp.pending) != 0 {
		t.Error("pending move should be cleared after a room block")
	}
}

func TestMapperPlacesUpDownWithoutCollision(t *testing.T) {
	mp := newMapper()
	mp.arrive("Ground", []string{"up"})
	mp.noteCommand("up")
	mp.arrive("Loft", []string{"down"})

	if mp.rooms["Ground"].pos == mp.rooms["Loft"].pos {
		t.Error("up/down rooms should not occupy the same cell")
	}
}

func TestMapRendersOnlyReciprocalConnectors(t *testing.T) {
	mp := newMapper()
	// A at origin with an east exit; B sits east of A but has no way back west.
	mp.rooms["A"] = &mroom{title: "A", pos: coord{0, 0}, exits: map[string]bool{"east": true}}
	mp.occupied[coord{0, 0}] = "A"
	mp.rooms["B"] = &mroom{title: "B", pos: coord{1, 0}, exits: map[string]bool{}}
	mp.occupied[coord{1, 0}] = "B"
	mp.cur = "A"

	if view := ansi.Strip(mp.render(11, 7)); strings.Contains(view, "─") {
		t.Errorf("a one-way / phantom adjacency should not draw a connector:\n%s", view)
	}

	// Give B a reciprocal west exit; now the connector is real.
	mp.rooms["B"].exits["west"] = true
	if view := ansi.Strip(mp.render(11, 7)); !strings.Contains(view, "─") {
		t.Errorf("reciprocal exits should draw a connector:\n%s", view)
	}
}

func TestMapShowsCurrentRoomExitStubs(t *testing.T) {
	mp := newMapper()
	mp.arrive("Lone Room", []string{"north", "east"}) // exits with no explored neighbors

	view := ansi.Strip(mp.render(11, 7))
	if !strings.Contains(view, "╵") {
		t.Errorf("an unexplored north exit should show a stub; got:\n%s", view)
	}
	if !strings.Contains(view, "╶") {
		t.Errorf("an unexplored east exit should show a stub; got:\n%s", view)
	}
	// A direction with no exit gets no stub.
	if strings.Contains(view, "╴") {
		t.Errorf("there is no west exit, so no west stub should appear; got:\n%s", view)
	}
}

func TestMapperVerticalExitGlyph(t *testing.T) {
	mp := newMapper()
	mp.arrive("Cellar", []string{"up"}) // current room with a vertical exit
	if view := ansi.Strip(mp.render(11, 7)); !strings.Contains(view, "◈") {
		t.Errorf("current room with up/down exit should render ◈; got:\n%s", view)
	}

	mp.noteCommand("north")
	mp.arrive("Hall", []string{"south"}) // Cellar is no longer current
	if view := ansi.Strip(mp.render(11, 7)); !strings.Contains(view, "◊") {
		t.Errorf("non-current room with up/down exit should render ◊; got:\n%s", view)
	}
}
