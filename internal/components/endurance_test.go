package components

import (
	"testing"
	"time"
)

func TestEnduranceSpend(t *testing.T) {
	e := NewEndurance(1)
	max := e.Max
	if max <= 0 || e.Current != max {
		t.Fatalf("new endurance = %d/%d, want full", e.Current, e.Max)
	}

	const cost = 25
	if !e.Spend(cost) || e.Current != max-cost {
		t.Fatalf("spend %d → %d, want %d", cost, e.Current, max-cost)
	}
	// Overspending fails and leaves the pool untouched.
	if e.Spend(max * 10) {
		t.Error("spending more than available should fail")
	}
	if e.Current != max-cost {
		t.Errorf("a failed spend must not change endurance; got %d", e.Current)
	}
}

func TestEnduranceRegen(t *testing.T) {
	e := NewEndurance(1)
	e.Current = e.Max - 20
	e.LastRegen = time.Now().Add(-10 * enduranceRegenInterval) // 10 points' worth elapsed
	e.Regen()
	if e.Current != e.Max-10 {
		t.Errorf("regen after 10 intervals → %d, want %d", e.Current, e.Max-10)
	}

	// Regen never overshoots Max.
	e.Current = e.Max - 1
	e.LastRegen = time.Now().Add(-100 * enduranceRegenInterval)
	e.Regen()
	if e.Current != e.Max {
		t.Errorf("regen should clamp to max; got %d/%d", e.Current, e.Max)
	}
}

func TestEnduranceRestore(t *testing.T) {
	e := NewEndurance(1)
	e.Current = e.Max - 50
	if got := e.Restore(30); got != 30 || e.Current != e.Max-20 {
		t.Errorf("restore 30 → gained %d, current %d", got, e.Current)
	}
	if got := e.Restore(1000); got != 20 || e.Current != e.Max {
		t.Errorf("restore should clamp to max; gained %d, current %d", got, e.Current)
	}
}

// TestConsumablesHaveTemplates guards against a registry entry whose item ID has
// no matching template (a typo would make the item un-spawnable).
func TestConsumablesHaveTemplates(t *testing.T) {
	for id := range ConsumableRegistry {
		if _, ok := ItemTemplates[id]; !ok {
			t.Errorf("consumable %q has no ItemTemplate", id)
		}
	}
}
