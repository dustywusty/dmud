package components

import (
	"sync"
	"time"
)

type StatusEffectType int

const (
	StatusEffectGuardBlessing StatusEffectType = iota
)

type StatusEffect struct {
	Type       StatusEffectType
	Name       string
	AppliedAt  time.Time
	Duration   time.Duration
	HPBonus    int
	Applied    bool
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

func (se *StatusEffects) GetEffect(effectType StatusEffectType) (*StatusEffect, bool) {
	se.RLock()
	defer se.RUnlock()

	for i := range se.Effects {
		if se.Effects[i].Type == effectType && !se.isExpired(se.Effects[i]) {
			return &se.Effects[i], true
		}
	}
	return nil, false
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
