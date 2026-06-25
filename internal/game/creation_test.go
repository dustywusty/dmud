package game

import (
	"testing"

	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/util"
)

// newGhostRig builds an un-manifested ghost (Created=false + a Creation tracker)
// with the components the creation flow touches.
func newGhostRig(t *testing.T) (*Game, *components.Player, ecs.Entity) {
	t.Helper()
	chdirToRepoRoot(t)
	world := ecs.NewWorld()

	area := &components.Area{Description: "The Veil"}
	client := newFakeClient("t:0")
	player := &components.Player{
		Client: client, Name: "Wisp", Area: area,
		CommandHistory: components.NewCommandHistory(), AutoComplete: util.NewAutoComplete(),
		EnteredWorld: true, Created: false,
	}
	g := &Game{world: world, players: make(map[string]*ecs.Entity)}

	pe := ecs.NewEntity()
	world.AddEntity(pe)
	world.AddComponent(&pe, player)
	world.AddComponent(&pe, components.NewExperience())
	world.AddComponent(&pe, &components.Health{Current: 100, Max: 100})
	world.AddComponent(&pe, components.NewStats())
	world.AddComponent(&pe, components.NewCreation())
	g.players[player.Name] = &pe

	return g, player, pe
}

func TestGhostCreationFlowAndManifest(t *testing.T) {
	g, player, pe := newGhostRig(t)

	// Name first.
	g.HandleRename(player, "Gandalf")
	if cr := g.getCreation(pe.ID); cr == nil || !cr.NamePicked {
		t.Fatal("naming should mark the name step")
	}
	if player.Created {
		t.Fatal("must not manifest after only a name")
	}

	// Then race (elf: WIS +4 → 14).
	g.handleRace(player, []string{"elf"}, g)
	if cr := g.getCreation(pe.ID); cr == nil || !cr.RacePicked {
		t.Fatal("choosing a race should mark the race step")
	}
	if player.Created {
		t.Fatal("must not manifest after name + race")
	}

	// Then class → manifests.
	g.handleClass(player, []string{"druid"}, g)
	if !player.Created {
		t.Fatal("should manifest once name + race + class are chosen")
	}
	if g.getCreation(pe.ID) != nil {
		t.Error("the Creation tracker should be removed on manifest")
	}

	// Affinity: Druid's primary (WIS) gets +classAffinityBonus on top of the elf base.
	stats := g.getStats(pe.ID)
	if got, want := stats.Get(components.WIS), 14+classAffinityBonus; got != want {
		t.Errorf("WIS after manifest = %d, want %d (elf 14 + affinity)", got, want)
	}
	if stats.Class != "druid" {
		t.Errorf("class = %q, want druid", stats.Class)
	}
}

// TestRaceAfterClassKeepsAffinity proves creation choices are order-independent:
// picking class then race preserves the class and its affinity.
func TestRaceAfterClassKeepsAffinity(t *testing.T) {
	g, player, pe := newGhostRig(t)

	g.handleClass(player, []string{"pyromancer"}, g) // INT primary, +5 affinity over human 10 = 15
	g.handleRace(player, []string{"ogre"}, g)        // ogre INT base = 10-8 = 2; affinity re-applied → 7
	stats := g.getStats(pe.ID)

	if stats.Class != "pyromancer" {
		t.Errorf("class lost across race reforge: %q", stats.Class)
	}
	if got, want := stats.Get(components.INT), 2+classAffinityBonus; got != want {
		t.Errorf("INT after reforge = %d, want %d (ogre 2 + affinity)", got, want)
	}
}

func TestGhostCommandGateList(t *testing.T) {
	// The creation/info commands are allowed; world actions are not.
	for _, allowed := range []string{"name", "race", "class", "look", "who", "help"} {
		if !ghostCommands[allowed] {
			t.Errorf("%q should be usable by a ghost", allowed)
		}
	}
	for _, blocked := range []string{"kill", "shift", "cast", "get", "north"} {
		if ghostCommands[blocked] {
			t.Errorf("%q should NOT be usable by a ghost", blocked)
		}
	}
}
