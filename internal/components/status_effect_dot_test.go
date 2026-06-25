package components

import (
	"testing"
	"time"
)

func TestStatusEffectTickWholeIntervals(t *testing.T) {
	se := NewStatusEffects()
	now := time.Now()
	se.AddEffect(StatusEffect{
		Type:         StatusEffectBurning,
		Name:         "Burning",
		AppliedAt:    now.Add(-5 * time.Second), // started 5s ago
		Duration:     10 * time.Second,
		TickHP:       -3,
		TickInterval: 2 * time.Second,
	})

	// 5s elapsed at a 2s interval => 2 whole ticks (2s, 4s), not 2.5.
	ticks := se.Tick(now)
	if len(ticks) != 1 {
		t.Fatalf("want 1 ticking effect, got %d", len(ticks))
	}
	if ticks[0].HPDelta != -6 {
		t.Errorf("want -6 (2 ticks x -3), got %d", ticks[0].HPDelta)
	}
	// Ticking again at the same instant must not re-apply (LastTick advanced).
	if again := se.Tick(now); len(again) != 0 {
		t.Errorf("want no further ticks at the same instant, got %v", again)
	}
}

func TestStatusEffectExpiredDoesNotTick(t *testing.T) {
	se := NewStatusEffects()
	se.AddEffect(StatusEffect{
		Type:         StatusEffectPoisoned,
		Name:         "Poison",
		AppliedAt:    time.Now().Add(-time.Hour),
		Duration:     time.Minute,
		TickHP:       -2,
		TickInterval: time.Second,
	})
	if ticks := se.Tick(time.Now()); len(ticks) != 0 {
		t.Errorf("expired DoT must not tick, got %v", ticks)
	}
}

func TestStatusEffectSnapshot(t *testing.T) {
	se := NewStatusEffects()
	now := time.Now()
	se.AddEffect(StatusEffect{Type: StatusEffectBlessed, Name: "Blessing", AppliedAt: now, Duration: 30 * time.Second, HPBonus: 10})
	se.AddEffect(StatusEffect{Type: StatusEffectBurning, Name: "Burning", AppliedAt: now, Duration: 6 * time.Second, TickHP: -4, TickInterval: 2 * time.Second})
	se.AddEffect(StatusEffect{Type: StatusEffectCharmed, Name: "Charm", AppliedAt: now.Add(-time.Hour), Duration: time.Minute}) // expired

	views := se.Snapshot()
	if len(views) != 2 {
		t.Fatalf("want 2 active effects (expired filtered out), got %d", len(views))
	}
	byName := map[string]EffectView{}
	for _, v := range views {
		byName[v.Name] = v
	}
	if v := byName["Blessing"]; v.Kind != "buff" || v.Magnitude != 10 {
		t.Errorf("blessing view = %+v, want kind=buff magnitude=10", v)
	}
	if v := byName["Burning"]; v.Kind != "dot" || v.Magnitude != -4 || v.Remaining <= 0 {
		t.Errorf("burning view = %+v, want kind=dot magnitude=-4 remaining>0", v)
	}
}

func TestStatusEffectKind(t *testing.T) {
	cases := map[StatusEffectType]string{
		StatusEffectBurning:       "dot",
		StatusEffectPoisoned:      "dot",
		StatusEffectRegenerating:  "heal",
		StatusEffectSlowed:        "control",
		StatusEffectStunned:       "control",
		StatusEffectBlessed:       "buff",
		StatusEffectGuardBlessing: "buff",
	}
	for typ, want := range cases {
		if got := typ.Kind(); got != want {
			t.Errorf("Kind(%d) = %q, want %q", typ, got, want)
		}
	}
}
