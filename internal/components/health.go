package components

import "sync"

type HealthComponent struct {
	sync.RWMutex

	CurrentHealth int
	MaxHealth     int
}

func (hc *HealthComponent) Heal(amount int) {
	hc.CurrentHealth += amount
	if hc.CurrentHealth > hc.MaxHealth {
		hc.CurrentHealth = hc.MaxHealth
	}
}

func (hc *HealthComponent) TakeDamage(amount int) {
	hc.CurrentHealth -= amount
	if hc.CurrentHealth < 0 {
		hc.CurrentHealth = 0
	}
}
