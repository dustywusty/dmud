package components

import (
	"math"
	"time"
)

// enduranceRegenInterval is how long one point of endurance takes to come back.
// Tune this to make the pool feel more or less punishing.
const enduranceRegenInterval = 2 * time.Second

// Endurance is a regenerating resource that gates active abilities (spells,
// later other exertions) so they can't be spammed. Like Health it is only
// touched on the game-loop goroutine, so it carries no mutex. Regen is lazy and
// time-based (see Regen), mirroring Mount — no dedicated system tick needed.
type Endurance struct {
	Current   int
	Max       int
	LastRegen time.Time
}

func (e *Endurance) Type() string { return "Endurance" }

// EnduranceBaseForLevel is the base endurance pool at a level, before any
// Constitution bonus. Exposed so the ceiling can be recomputed as you level or
// train CON (mirroring how Health's effective max is derived).
func EnduranceBaseForLevel(level int) int {
	return int(math.Floor(100 * GetLevelScaling(level)))
}

// NewEndurance gives a generous starting pool that scales with level, the same
// way Health does. Constitution adds to it later (see EnduranceBonus).
func NewEndurance(level int) *Endurance {
	max := EnduranceBaseForLevel(level)
	return &Endurance{Current: max, Max: max, LastRegen: time.Now()}
}

// Regen lazily restores endurance based on the time elapsed since the last
// accounting. Call it before reading Current so the value is always up to date.
func (e *Endurance) Regen() {
	if e.Max <= 0 {
		return
	}
	if e.LastRegen.IsZero() || e.Current >= e.Max {
		e.LastRegen = time.Now()
		if e.Current > e.Max {
			e.Current = e.Max
		}
		return
	}
	gained := int(time.Since(e.LastRegen) / enduranceRegenInterval)
	if gained <= 0 {
		return
	}
	e.Current += gained
	if e.Current > e.Max {
		e.Current = e.Max
	}
	e.LastRegen = e.LastRegen.Add(time.Duration(gained) * enduranceRegenInterval)
}

// Spend deducts cost after catching up regen. It returns false (changing
// nothing) when there isn't enough endurance, so callers can refuse the action.
func (e *Endurance) Spend(cost int) bool {
	e.Regen()
	if cost <= 0 {
		return true
	}
	if e.Current < cost {
		return false
	}
	e.Current -= cost
	return true
}

// Restore adds endurance up to Max (for consumables) and returns the amount
// actually restored.
func (e *Endurance) Restore(amount int) int {
	e.Regen()
	if amount <= 0 {
		return 0
	}
	before := e.Current
	e.Current += amount
	if e.Current > e.Max {
		e.Current = e.Max
	}
	return e.Current - before
}
