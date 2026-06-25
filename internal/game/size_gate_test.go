package game

import (
	"testing"

	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/systems"
	"dmud/internal/util"
)

// TestSizeBlocksNarrowExit proves a Large creature can't squeeze through an exit
// capped at Medium, while a Medium one passes.
func TestSizeBlocksNarrowExit(t *testing.T) {
	chdirToRepoRoot(t)
	world := ecs.NewWorld()

	roomA := &components.Area{Description: "Cave Mouth"}
	roomB := &components.Area{Description: "Tight Crawl"}
	roomA.Exits = []components.Exit{{Direction: "north", Area: roomB, MaxSize: components.SizeMedium}}

	moveFrom := func(race string) *components.Area {
		client := newFakeClient("t:0")
		player := &components.Player{
			Client: client, Name: "Mover-" + race, Area: roomA,
			CommandHistory: components.NewCommandHistory(), AutoComplete: util.NewAutoComplete(), EnteredWorld: true,
		}
		pe := ecs.NewEntity()
		world.AddEntity(pe)
		world.AddComponent(&pe, player)
		world.AddComponent(&pe, &components.Health{Current: 100, Max: 100})
		world.AddComponent(&pe, components.NewStatsForRace(race))
		world.AddComponent(&pe, &components.Movement{Direction: "north", Status: components.Walking})
		systems.HandleMovement(world, pe)
		return player.Area
	}

	if got := moveFrom("human"); got != roomB {
		t.Errorf("a Medium human should fit the narrow exit; ended in %q", got.Description)
	}
	if got := moveFrom("ogre"); got != roomA {
		t.Errorf("a Large ogre should be blocked; ended in %q", got.Description)
	}
}
