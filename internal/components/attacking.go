package components

import "dmud/internal/common"

type AttackingComponent struct {
	TargetID  common.EntityID
	MinDamage int
	MaxDamage int
}
