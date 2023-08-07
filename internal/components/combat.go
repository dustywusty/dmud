package components

import (
	"dmud/internal/common"
	"sync"
)

type CombatComponent struct {
	sync.RWMutex
	TargetID  common.EntityID
	MinDamage int
	MaxDamage int
}
