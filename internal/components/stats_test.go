package components

import "testing"

func approxEq(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d < 1e-9
}

func TestStatsTrainDiminishesAndCaps(t *testing.T) {
	s := NewStats()
	if s.Get(STR) != statBase {
		t.Fatalf("new stat = %d, want %d", s.Get(STR), statBase)
	}

	// A capped stat never rises, no matter how often it's used.
	s.Set(STR, statCap)
	for i := 0; i < 2000; i++ {
		if raised, _ := s.Train(STR); raised {
			t.Fatal("a capped stat must not rise")
		}
	}
	if s.Get(STR) != statCap {
		t.Errorf("capped stat changed to %d", s.Get(STR))
	}

	// A base stat rises within a bounded number of uses (statistically certain at
	// an ~18% per-use chance).
	s2 := NewStats()
	rose := false
	for i := 0; i < 1000; i++ {
		if raised, _ := s2.Train(DEX); raised {
			rose = true
			break
		}
	}
	if !rose {
		t.Error("a base stat should rise within 1000 uses")
	}
}

func TestStatsFactors(t *testing.T) {
	s := NewStats()
	if !approxEq(s.MeleeFactor(), 1.0) || !approxEq(s.ArcaneFactor(), 1.0) || !approxEq(s.HealFactor(), 1.0) {
		t.Errorf("base factors should be 1.0; got melee=%v arcane=%v heal=%v", s.MeleeFactor(), s.ArcaneFactor(), s.HealFactor())
	}
	if s.HPBonus() != 0 || !approxEq(s.CostFactor(), 1.0) {
		t.Errorf("base HP bonus / cost factor wrong: %d / %v", s.HPBonus(), s.CostFactor())
	}

	s.Set(STR, 60)
	if !approxEq(s.MeleeFactor(), 1.5) {
		t.Errorf("STR 60 melee factor = %v, want 1.5", s.MeleeFactor())
	}
	s.Set(CON, 20)
	if s.HPBonus() != 30 {
		t.Errorf("CON 20 HP bonus = %d, want 30", s.HPBonus())
	}
	s.Set(DEX, 60)
	if !approxEq(s.CostFactor(), 0.8) {
		t.Errorf("DEX 60 cost factor = %v, want 0.8", s.CostFactor())
	}

	// Set clamps to the [min, cap] band (min is below base so races can penalize).
	s.Set(WIS, 9999)
	if s.Get(WIS) != statCap {
		t.Errorf("Set above cap = %d, want %d", s.Get(WIS), statCap)
	}
	s.Set(WIS, -5)
	if s.Get(WIS) != statMin {
		t.Errorf("Set below min = %d, want %d", s.Get(WIS), statMin)
	}
}
