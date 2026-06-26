package game

import (
	"strings"
	"testing"

	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/util"
)

// newClericRig builds a wounded caster with a given Wisdom, for heal-scaling tests.
func newClericRig(t *testing.T, wis int) (*Game, *components.Player, *components.Health, *fakeClient) {
	t.Helper()
	chdirToRepoRoot(t)
	world := ecs.NewWorld()

	area := &components.Area{Description: "Sickbay"}
	client := newFakeClient("t:0")
	player := &components.Player{
		Client: client, Name: "Cleric", Area: area,
		CommandHistory: components.NewCommandHistory(), AutoComplete: util.NewAutoComplete(), EnteredWorld: true,
	}
	g := &Game{world: world, players: make(map[string]*ecs.Entity)}

	pe := ecs.NewEntity()
	world.AddEntity(pe)
	world.AddComponent(&pe, player)
	world.AddComponent(&pe, components.NewExperience()) // level 1
	health := &components.Health{Current: 50, Max: 200}
	world.AddComponent(&pe, health)
	stats := components.NewStats()
	stats.Set(components.WIS, wis)
	world.AddComponent(&pe, stats)
	g.players[player.Name] = &pe

	return g, player, health, client
}

func TestHealScalesWithWisdom(t *testing.T) {
	// mend at level 1 = base 8 + 1*4 = 12, then ×Wisdom heal factor.
	heal := func(wis int) int {
		g, player, health, _ := newClericRig(t, wis)
		before := health.Current
		g.castHealSpell(player, nil, healSpellSpec{name: "mend", base: 8, perLevel: 4})
		return health.Current - before
	}

	base := heal(10) // factor 1.0
	high := heal(60) // factor 1.5

	if base != 12 {
		t.Errorf("WIS 10 mend healed %d, want 12", base)
	}
	if high != 18 {
		t.Errorf("WIS 60 mend healed %d, want 18", high)
	}
	if high <= base {
		t.Errorf("higher Wisdom should heal more: %d vs %d", high, base)
	}
}

func TestScoreCommand(t *testing.T) {
	g, player, _, client := newClericRig(t, 10)
	client.drain()

	g.handleScore(player, nil, g)
	out := client.drain()

	for _, want := range []string{"STR", "DEX", "CON", "INT", "WIS", "max HP", "healing", "melee damage"} {
		if !strings.Contains(out, want) {
			t.Errorf("score output missing %q\n--- output ---\n%s", want, out)
		}
	}
}
