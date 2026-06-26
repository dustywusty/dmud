package components

import (
	"dmud/internal/common"
	"sync"
	"time"
)

type StatusEffectType int

const (
	StatusEffectGuardBlessing StatusEffectType = iota
	StatusEffectControlledUndead
	StatusEffectCharmed
	StatusEffectBlessed
	// Appended (persisted as ints — never reorder the values above):
	StatusEffectBurning      // damage over time (fire)
	StatusEffectPoisoned     // damage over time (poison)
	StatusEffectRegenerating // heal over time
	StatusEffectSlowed       // control
	StatusEffectStunned      // control
	StatusEffectInvigorated  // endurance regen over time
)

// Kind buckets an effect for the client UI: "buff", "dot", "heal", or "control".
func (t StatusEffectType) Kind() string {
	switch t {
	case StatusEffectBurning, StatusEffectPoisoned:
		return "dot"
	case StatusEffectRegenerating, StatusEffectInvigorated:
		return "heal"
	case StatusEffectSlowed, StatusEffectStunned:
		return "control"
	default:
		return "buff"
	}
}

type StatusEffect struct {
	Type      StatusEffectType
	Name      string
	AppliedAt time.Time
	Duration  time.Duration
	HPBonus   int
	Applied   bool

	// Periodic HP change: <0 damages (DoT), >0 heals (regen), 0 = none. Applied
	// once per TickInterval by the StatusEffectSystem.
	TickHP int
	// Periodic endurance change (>0 restores), on the same TickInterval/LastTick.
	TickEP       int
	TickInterval time.Duration
	LastTick     time.Time

	SourceEntityID      common.EntityID
	SuppressAggro       bool
	SuppressRetaliation bool
}

// Remaining is the time left before the effect expires (0 = permanent/over).
func (e StatusEffect) Remaining() time.Duration {
	if e.Duration == 0 {
		return 0
	}
	if rem := e.Duration - time.Since(e.AppliedAt); rem > 0 {
		return rem
	}
	return 0
}

type StatusEffects struct {
	sync.RWMutex
	Effects []StatusEffect
}

func NewStatusEffects() *StatusEffects {
	return &StatusEffects{
		Effects: make([]StatusEffect, 0),
	}
}

func (se *StatusEffects) AddEffect(effect StatusEffect) {
	se.Lock()
	defer se.Unlock()

	for i, existing := range se.Effects {
		if existing.Type == effect.Type {
			se.Effects[i] = effect
			return
		}
	}

	se.Effects = append(se.Effects, effect)
}

func (se *StatusEffects) HasEffect(effectType StatusEffectType) bool {
	se.RLock()
	defer se.RUnlock()

	for _, effect := range se.Effects {
		if effect.Type == effectType && !se.isExpired(effect) {
			return true
		}
	}
	return false
}

func (se *StatusEffects) GetEffect(effectType StatusEffectType) (StatusEffect, bool) {
	se.RLock()
	defer se.RUnlock()

	for i := range se.Effects {
		if se.Effects[i].Type == effectType && !se.isExpired(se.Effects[i]) {
			return se.Effects[i], true
		}
	}
	return StatusEffect{}, false
}

func (se *StatusEffects) RemoveExpired() []StatusEffect {
	se.Lock()
	defer se.Unlock()

	var removed []StatusEffect
	var active []StatusEffect

	for _, effect := range se.Effects {
		if se.isExpired(effect) {
			removed = append(removed, effect)
		} else {
			active = append(active, effect)
		}
	}

	se.Effects = active
	return removed
}

func (se *StatusEffects) isExpired(effect StatusEffect) bool {
	if effect.Duration == 0 {
		return false
	}
	return time.Since(effect.AppliedAt) >= effect.Duration
}

// EffectTick is one effect's accumulated periodic HP change since it last ticked.
type EffectTick struct {
	Name    string
	HPDelta int // <0 damage, >0 heal
	EPDelta int // >0 restores endurance
	Source  common.EntityID
}

// Tick advances every periodic (DoT/regen) effect to now and returns the HP
// changes to apply. Whole intervals only, so a laggy loop catches up exactly
// rather than over- or under-applying.
func (se *StatusEffects) Tick(now time.Time) []EffectTick {
	se.Lock()
	defer se.Unlock()

	var out []EffectTick
	for i := range se.Effects {
		e := &se.Effects[i]
		if (e.TickHP == 0 && e.TickEP == 0) || e.TickInterval <= 0 || se.isExpired(*e) {
			continue
		}
		if e.LastTick.IsZero() {
			e.LastTick = e.AppliedAt
		}
		n := int(now.Sub(e.LastTick) / e.TickInterval)
		if n <= 0 {
			continue
		}
		e.LastTick = e.LastTick.Add(time.Duration(n) * e.TickInterval)
		out = append(out, EffectTick{Name: e.Name, HPDelta: e.TickHP * n, EPDelta: e.TickEP * n, Source: e.SourceEntityID})
	}
	return out
}

// EffectView is the client-facing snapshot of one active effect.
type EffectView struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Remaining int    `json:"remaining"`           // seconds; 0 = permanent
	Magnitude int    `json:"magnitude,omitempty"` // HP/tick (DoT/regen) or HP bonus (buff)
}

// Snapshot returns the active effects for a char.vitals push (buff/debuff timers).
func (se *StatusEffects) Snapshot() []EffectView {
	se.RLock()
	defer se.RUnlock()

	var out []EffectView
	for _, e := range se.Effects {
		if se.isExpired(e) {
			continue
		}
		mag := e.HPBonus
		if e.TickHP != 0 {
			mag = e.TickHP
		}
		if e.TickEP != 0 {
			mag = e.TickEP
		}
		out = append(out, EffectView{
			Name:      e.Name,
			Kind:      e.Type.Kind(),
			Remaining: int(e.Remaining().Seconds()),
			Magnitude: mag,
		})
	}
	return out
}

func (se *StatusEffects) GetTotalHPBonus() int {
	se.RLock()
	defer se.RUnlock()

	total := 0
	for _, effect := range se.Effects {
		if !se.isExpired(effect) {
			total += effect.HPBonus
		}
	}
	return total
}

func (se *StatusEffects) HasSuppressedAggro() bool {
	se.RLock()
	defer se.RUnlock()

	for _, effect := range se.Effects {
		if !se.isExpired(effect) && effect.SuppressAggro {
			return true
		}
	}

	return false
}

func (se *StatusEffects) HasSuppressedRetaliation() bool {
	se.RLock()
	defer se.RUnlock()

	for _, effect := range se.Effects {
		if !se.isExpired(effect) && effect.SuppressRetaliation {
			return true
		}
	}

	return false
}

// ControlledBy reports whether this creature is under an active charm or control
// effect cast by the given master entity — i.e. it should follow that master
// around. Expired effects and effects from other casters don't count.
func (se *StatusEffects) ControlledBy(master common.EntityID) bool {
	se.RLock()
	defer se.RUnlock()

	for _, effect := range se.Effects {
		if se.isExpired(effect) || effect.SourceEntityID != master {
			continue
		}
		if effect.Type == StatusEffectCharmed || effect.Type == StatusEffectControlledUndead {
			return true
		}
	}
	return false
}

func (se *StatusEffects) GetActiveSuppressionEffect() (StatusEffect, bool) {
	se.RLock()
	defer se.RUnlock()

	for _, effect := range se.Effects {
		if se.isExpired(effect) {
			continue
		}
		if effect.SuppressAggro || effect.SuppressRetaliation {
			return effect, true
		}
	}

	return StatusEffect{}, false
}
