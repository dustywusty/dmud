package components

import (
	"math"
	"sync"
)

type HealthStatus int

const (
	Healthy HealthStatus = iota
	Injured
	Dead
)

type Health struct {
	sync.RWMutex

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
