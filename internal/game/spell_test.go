package game

import (
	"strings"
	"testing"

	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/util"
)

type spellRig struct {
	g            *Game
	player       *components.Player
	client       *fakeClient
	end          *components.Endurance
	exp          *components.Experience
	targetHealth *components.Health
}

// newSpellRig builds a Game with a caster at the given level (with endurance and
// experience) standing next to a high-HP goblin dummy to cast at.
func newSpellRig(t *testing.T, casterLevel int) *spellRig {
	t.Helper()
	chdirToRepoRoot(t)

	world := ecs.NewWorld()
	areaComp, err := world.GetComponent("201", "Area")
	if err != nil {
		t.Fatalf("area 201 not loaded: %v", err)
	}
	area := areaComp.(*components.Area)

	g := &Game{world: world, players: make(map[string]*ecs.Entity)}

	targetEnt := ecs.NewEntity()
	world.AddEntity(targetEnt)
	world.AddComponent(&targetEnt, &components.NPC{Name: "a sneaky goblin", TemplateID: "goblin", Area: area})
	targetHealth := &components.Health{Current: 500, Max: 500}
	world.AddComponent(&targetEnt, targetHealth)

	client := newFakeClient("test:0")
	player := &components.Player{
		Client: client, Name: "Mage", Area: area,
		CommandHistory: components.NewCommandHistory(), AutoComplete: util.NewAutoComplete(), EnteredWorld: true,
	}
	exp := components.NewExperience()
	exp.Level = casterLevel
	end := components.NewEndurance(casterLevel)

	pe := ecs.NewEntity()
	world.AddEntity(pe)
	world.AddComponent(&pe, player)
	world.AddComponent(&pe, exp)
	world.AddComponent(&pe, end)
	g.players[player.Name] = &pe

	return &spellRig{g: g, player: player, client: client, end: end, exp: exp, targetHealth: targetHealth}
}

func TestSpellLevelGate(t *testing.T) {
	r := newSpellRig(t, 1) // firebolt needs level 3
	r.client.drain()

	epBefore := r.end.Current
	handleCast(r.player, []string{"firebolt", "goblin"}, r.g)
	out := r.client.drain()

	if !strings.Contains(out, "requires level 3") {
		t.Errorf("a level-1 caster should be refused firebolt; got %q", out)
	}
	if r.targetHealth.Current != 500 {
		t.Errorf("a level-gated refusal must not deal damage; hp=%d", r.targetHealth.Current)
	}
	if r.end.Current != epBefore {
		t.Errorf("a level-gated refusal must not spend endurance; %d→%d", epBefore, r.end.Current)
	}
}

func TestSpellDamageAndCost(t *testing.T) {
	r := newSpellRig(t, 5) // meets firebolt's level 3
	maxEP := r.end.Max
	r.client.drain()

	handleCast(r.player, []string{"firebolt", "goblin"}, r.g)
	out := r.client.drain()

	if r.targetHealth.Current >= 500 {
		t.Errorf("firebolt should damage the goblin; hp=%d", r.targetHealth.Current)
	}
	if r.end.Current != maxEP-16 {
		t.Errorf("firebolt should cost 16 endurance; ep=%d, want %d", r.end.Current, maxEP-16)
	}
	if !strings.Contains(out, "firebolt") {
		t.Errorf("expected a firebolt hit message; got %q", out)
	}
}

func TestSpellExhausted(t *testing.T) {
	r := newSpellRig(t, 5)
	r.end.Current = 5 // below firebolt's 16
	r.client.drain()

	handleCast(r.player, []string{"firebolt", "goblin"}, r.g)
	out := r.client.drain()

	if !strings.Contains(strings.ToLower(out), "exhausted") {
		t.Errorf("an exhausted caster should be refused; got %q", out)
	}
	if r.targetHealth.Current != 500 {
		t.Errorf("an exhausted refusal must not deal damage; hp=%d", r.targetHealth.Current)
	}
}

func TestSpellKillGrantsXP(t *testing.T) {
	r := newSpellRig(t, 20)
	r.targetHealth.Current = 5 // immolate will one-shot it
	xpBefore := r.exp.Current
	r.client.drain()

	handleCast(r.player, []string{"immolate", "goblin"}, r.g)
	out := r.client.drain()

	if !strings.Contains(out, "defeated") && !strings.Contains(out, "slain") {
		t.Errorf("a lethal spell should report a kill; got %q", out)
	}
	if r.exp.Current <= xpBefore {
		t.Errorf("killing with a spell should award XP; %d→%d", xpBefore, r.exp.Current)
	}
}

func TestSpellbookListing(t *testing.T) {
	r := newSpellRig(t, 1)
	r.client.drain()
	r.g.handleSpells(r.player, nil, r.g)
	out := r.client.drain()

	for _, want := range []string{"Pyromancy", "spark", "firebolt", "(locked)", "Restoration", "heal"} {
		if !strings.Contains(out, want) {
			t.Errorf("spellbook missing %q\n--- output ---\n%s", want, out)
		}
	}
}

func TestPyromancyRegistered(t *testing.T) {
	cases := map[string]struct{ minLevel, cost int }{
		"spark":    {1, 8},
		"firebolt": {3, 16},
		"immolate": {15, 60},
	}
	for name, want := range cases {
		def := spellRegistry[name]
		if def == nil {
			t.Errorf("spell %q not registered", name)
			continue
		}
		if def.School != "pyromancy" {
			t.Errorf("%s school = %q, want pyromancy", name, def.School)
		}
		if def.MinLevel != want.minLevel || def.Cost != want.cost {
			t.Errorf("%s = L%d/%dEP, want L%d/%dEP", name, def.MinLevel, def.Cost, want.minLevel, want.cost)
		}
	}
}
