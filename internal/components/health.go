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

	Current int
	Max     int
	Status  HealthStatus
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
