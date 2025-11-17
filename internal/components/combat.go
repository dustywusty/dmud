package components

import (
	"dmud/internal/common"
	"sync"
)

type Combat struct {
	sync.RWMutex

	TargetID    common.EntityID
	TargetQueue []common.EntityID // Queue of additional targets to attack
	MinDamage   int
	MaxDamage   int
}
