package game

import (
	"testing"
	"time"

	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/systems"
	"dmud/internal/util"
)

// TestCharmedNPCFollows proves a charmed minion moves with its master, while an
// uncontrolled NPC in the same room stays behind.
func TestCharmedNPCFollows(t *testing.T) {
	chdirToRepoRoot(t)
	world := ecs.NewWorld()

	areaA := &components.Area{Description: "Cinder Lane"}
	areaB := &components.Area{Description: "Ash Court"}
	areaA.Exits = []components.Exit{{Direction: "north", Area: areaB}}

	client := newFakeClient("t:0")
	player := &components.Player{
		Client: client, Name: "Mage", Area: areaA,
		CommandHistory: components.NewCommandHistory(), AutoComplete: util.NewAutoComplete(), EnteredWorld: true,
	}
	pe := ecs.NewEntity()
	world.AddEntity(pe)
	world.AddComponent(&pe, player)
	world.AddComponent(&pe, &components.Health{Current: 100, Max: 100})

	// A goblin charmed by this player.
	charmed := &components.NPC{Name: "a charmed goblin", TemplateID: "goblin", Area: areaA}
	ce := ecs.NewEntity()
	world.AddEntity(ce)
	world.AddComponent(&ce, charmed)
	se := components.NewStatusEffects()
	se.AddEffect(components.StatusEffect{
		Type: components.StatusEffectCharmed, Name: "Charmed",
		AppliedAt: time.Now(), Duration: time.Minute, SourceEntityID: pe.ID,
	})
	world.AddComponent(&ce, se)

	// An unrelated goblin that should stay put.
	bystander := &components.NPC{Name: "a wild goblin", TemplateID: "goblin", Area: areaA}
	be := ecs.NewEntity()
	world.AddEntity(be)
	world.AddComponent(&be, bystander)

	// Walk north.
	world.AddComponent(&pe, &components.Movement{Direction: "north", Status: components.Walking})
	systems.HandleMovement(world, pe)

	if player.Area != areaB {
		t.Fatalf("player should be in Ash Court, got %q", player.Area.Description)
	}
	if charmed.Area != areaB {
		t.Errorf("charmed goblin should follow to Ash Court, got %q", charmed.Area.Description)
	}
	if bystander.Area != areaA {
		t.Errorf("an uncontrolled goblin must NOT follow; got %q", bystander.Area.Description)
	}
}
