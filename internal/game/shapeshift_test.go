package game

import (
	"strings"
	"testing"
)

func TestShapeshiftMeleeAndCastBlock(t *testing.T) {
	r := newSpellRig(t, 6) // level 6 unlocks every form
	r.client.drain()

	// Unarmed by default.
	if mn, mx := r.g.playerMeleeDamage(r.player); mn != 10 || mx != 50 {
		t.Errorf("unarmed melee = %d-%d, want 10-50", mn, mx)
	}

	// Shift to bear → melee becomes the bear's natural weapons.
	r.g.handleShift(r.player, []string{"bear"}, r.g)
	if mn, mx := r.g.playerMeleeDamage(r.player); mn != 14 || mx != 26 {
		t.Errorf("bear melee = %d-%d, want 14-26", mn, mx)
	}

	// Casting is blocked while shifted.
	r.client.drain()
	handleCast(r.player, []string{"moonfire", "goblin"}, r.g)
	if out := r.client.drain(); !strings.Contains(out, "beast form") {
		t.Errorf("casting while shifted should be refused; got %q", out)
	}

	// Revert → unarmed again, and casting works (damages the target).
	r.g.handleRevert(r.player, nil, r.g)
	if mn, mx := r.g.playerMeleeDamage(r.player); mn != 10 || mx != 50 {
		t.Errorf("after revert melee = %d-%d, want 10-50", mn, mx)
	}
	before := r.targetHealth.Current
	handleCast(r.player, []string{"moonfire", "goblin"}, r.g)
	if r.targetHealth.Current >= before {
		t.Error("after reverting, casting moonfire should damage the target again")
	}
}

func TestShapeshiftLevelGate(t *testing.T) {
	r := newSpellRig(t, 1) // panther needs level 6
	r.client.drain()

	r.g.handleShift(r.player, []string{"panther"}, r.g)
	if out := r.client.drain(); !strings.Contains(out, "requires level 6") {
		t.Errorf("a low-level panther shift should be gated; got %q", out)
	}
	if mn, mx := r.g.playerMeleeDamage(r.player); mn != 10 || mx != 50 {
		t.Errorf("a gated shift must not change melee; got %d-%d", mn, mx)
	}
}
