package components

import (
	"dmud/internal/common"
	"sync"
)

type AttackingComponent struct {
	sync.RWMutex
	TargetID  common.EntityID
	MinDamage int
	MaxDamage int
}
