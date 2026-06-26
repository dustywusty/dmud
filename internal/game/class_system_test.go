package game

import (
	"strings"
	"testing"

	"dmud/internal/components"
)

func TestClassPickAffinityGatingMacros(t *testing.T) {
	g, player, _, client := newClericRig(t, 10)
	entityID, _ := g.getPlayerEntity(player)
	stats := g.getStats(entityID)
	wisBefore := stats.Get(components.WIS)
	client.drain()

	g.handleClass(player, []string{"cleric"}, g)
	out := client.drain()

	if stats.Class != "cleric" {
		t.Errorf("class = %q, want cleric", stats.Class)
	}
	if got := stats.Get(components.WIS); got != wisBefore+classAffinityBonus {
		t.Errorf("WIS affinity not applied: %d -> %d (want +%d)", wisBefore, got, classAffinityBonus)
	}
	if !strings.Contains(out, "Wired") {
		t.Errorf("expected a macro-loadout confirmation; got %q", out)
	}

	// Gating: a Cleric casts restoration + holy, never pyromancy.
	if !classAllowsSchool(stats, "restoration") || !classAllowsSchool(stats, "holy") {
		t.Error("cleric should be allowed restoration + holy")
	}
	if classAllowsSchool(stats, "pyromancy") {
		t.Error("cleric must not be allowed pyromancy")
	}

	// Re-picking the same class doesn't re-grant affinity.
	wisNow := stats.Get(components.WIS)
	g.handleClass(player, []string{"cleric"}, g)
	if stats.Get(components.WIS) != wisNow {
		t.Errorf("re-picking a class must not stack affinity (%d -> %d)", wisNow, stats.Get(components.WIS))
	}
}

func TestClassGatingBlocksOffSchoolCast(t *testing.T) {
	g, player, _, client := newClericRig(t, 10)
	entityID, _ := g.getPlayerEntity(player)
	g.getStats(entityID).Class = "cleric"
	client.drain()

	handleCast(player, []string{"firebolt", "goblin"}, g)
	if out := client.drain(); !strings.Contains(out, "discipline") {
		t.Errorf("a cleric casting firebolt should be refused by class gating; got %q", out)
	}
}

func TestHolyAndShadowSchools(t *testing.T) {
	if len(spellsInSchool("holy")) == 0 || len(spellsInSchool("shadow")) == 0 {
		t.Fatal("holy and shadow schools should both have spells")
	}
	if d := spellRegistry["drain"]; d == nil || d.School != "shadow" {
		t.Errorf("drain mis-registered: %+v", d)
	}
	if b := spellRegistry["bless"]; b == nil || b.School != "holy" {
		t.Errorf("bless mis-registered: %+v", b)
	}
	cleric, _ := classByKey("cleric")
	warlock, _ := classByKey("warlock")
	if strings.Join(cleric.Schools, ",") != "restoration,holy" {
		t.Errorf("cleric schools = %v", cleric.Schools)
	}
	if strings.Join(warlock.Schools, ",") != "domination,shadow" {
		t.Errorf("warlock schools = %v", warlock.Schools)
	}
}

// TestCastDefaultsToCombatTarget proves a no-target damage cast hits whatever the
// player is currently fighting — the trick that makes one-press spell macros work.
func TestCastDefaultsToCombatTarget(t *testing.T) {
	r := newSpellRig(t, 5) // classless, has endurance + a goblin
	r.client.drain()

	// Find the goblin entity and engage it.
	ents, _ := r.g.world.FindEntitiesByComponentPredicate("NPC", func(i any) bool {
		_, ok := i.(*components.NPC)
		return ok
	})
	if len(ents) == 0 {
		t.Fatal("no NPC in rig")
	}
	pe := r.g.players[r.player.Name]
	r.g.world.AddComponent(pe, &components.Combat{TargetID: ents[0].ID, MinDamage: 1, MaxDamage: 1})

	before := r.targetHealth.Current
	handleCast(r.player, []string{"firebolt"}, r.g) // no explicit target
	if r.targetHealth.Current >= before {
		t.Errorf("a no-target cast should hit the current combat target; hp %d -> %d", before, r.targetHealth.Current)
	}
}
