package components

import (
	"dmud/internal/common"
)

// Combat is a transient component that requests and maintains an attack; it is
// processed by CombatSystem each tick. All access happens on the game loop
// goroutine (systems and command handlers), so it carries no mutex.
type Combat struct {
	TargetID    common.EntityID
	TargetQueue []common.EntityID // additional targets to attack after the current one
	MinDamage   int
	MaxDamage   int
}
