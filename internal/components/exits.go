package components

import . "dmud/internal/ecs"

type Exits interface {
	GetExits() map[string]*Entity
}

type ExitsComponent struct {
	Exits map[string]*Entity
}

func (e *ExitsComponent) GetExits() map[string]*Entity {
	return e.Exits
}
