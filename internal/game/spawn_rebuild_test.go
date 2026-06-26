package game

import (
	"testing"

	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
)

// TestRebuildSpawnTrackingTrimsAndReregisters guards the merchant/chicken
// duplication bug: after a restart, restored NPCs that had wandered out of their
// spawn area must still count toward that spawn (so it doesn't duplicate them),
// and any surplus over MaxCount is pruned (so an accumulated horde collapses).
func TestRebuildSpawnTrackingTrimsAndReregisters(t *testing.T) {
	world := ecs.NewWorld()
	g := &Game{world: world, players: make(map[string]*ecs.Entity)}

	home := &components.Area{}
	away := &components.Area{} // a different room the mob "wandered" to

	spawnEnt := ecs.NewEntity()
	world.AddEntity(spawnEnt)
	sp := components.NewSpawn(spawnEnt.ID)
	sp.Configs = []components.SpawnConfig{
		{Type: components.SpawnTypeNPC, TemplateID: "testmob", MinCount: 1, MaxCount: 1},
	}
	world.AddComponent(&spawnEnt, sp)

	// Three mobs of the managed template — two of them wandered to 'away'.
	var ids []common.EntityID
	for _, area := range []*components.Area{home, away, away} {
		e := ecs.NewEntity()
		world.AddEntity(e)
		world.AddComponent(&e, &components.NPC{Name: "testmob", TemplateID: "testmob", Area: area})
		ids = append(ids, e.ID)
	}

	g.rebuildSpawnTracking()

	if got := len(sp.ActiveSpawns["testmob"]); got != 1 {
		t.Fatalf("tracked %d mobs, want 1 (MaxCount) — wandered mobs must count and surplus must trim", got)
	}
	alive := 0
	for _, id := range ids {
		if _, err := world.FindEntity(id); err == nil {
			alive++
		}
	}
	if alive != 1 {
		t.Errorf("after trim, %d mobs remain in the world, want 1", alive)
	}
}
