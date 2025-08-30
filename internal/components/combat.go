package components

import (
	"dmud/internal/common"
	"sync"
)

type Combat struct {
	sync.RWMutex

	TargetID  common.EntityID
	MinDamage int
	MaxDamage int
}
