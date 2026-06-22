package components

import (
	"math"
)

type HealthStatus int

const (
	Healthy HealthStatus = iota
	Injured
	Dead
)

// Health is accessed only on the game loop goroutine (systems, command
// handlers, and the on-loop autosave), so it carries no mutex.
type Health struct {
	Current int
	Max     int
	Status  HealthStatus
}

func (hc *Health) GetEffectiveMax(bonus int) int {
	return hc.Max + bonus
}

func (hc *Health) Heal(amount int) {
	hc.Current += amount
	if hc.Current > hc.Max {
		hc.Current = hc.Max
		hc.Status = Healthy
	}
}

func (hc *Health) TakeDamage(amount int) {
	hc.Current -= amount
	if hc.Current < 1 {
		hc.Current = 0
		hc.Status = Dead
	} else {
		hc.Status = Injured
	}
}

func NewHealth(level int) *Health {
	baseHP := 100
	scaledHP := int(math.Floor(float64(baseHP) * GetLevelScaling(level)))
	return &Health{
		Current: scaledHP,
		Max:     scaledHP,
		Status:  Healthy,
	}
}
