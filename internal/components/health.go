package components

import "sync"

type HealthStatus int

const (
	Healthy HealthStatus = iota
	Injured
	Dead
)

type Health struct {
	sync.RWMutex

	CurrentHealth int
	MaxHealth     int
	Status        HealthStatus
}

func (hc *Health) Heal(amount int) {
	hc.CurrentHealth += amount
	if hc.CurrentHealth > hc.MaxHealth {
		hc.CurrentHealth = hc.MaxHealth
		hc.Status = Healthy
	}
}

func (hc *Health) TakeDamage(amount int) {
	hc.CurrentHealth -= amount
	if hc.CurrentHealth < 1 {
		hc.CurrentHealth = 0
		hc.Status = Dead
	} else {
		hc.Status = Injured
	}
}
