package systems

import (
	"dmud/internal/components"
	"dmud/internal/ecs"
	"fmt"
)

type CombatSystem struct{}

func (cs *CombatSystem) Update(w *ecs.World, deltaTime float64) {
	// Find all entities with an AttackingComponent
	attackingEntities, err := w.FindEntitiesByComponentPredicate("AttackingComponent", func(i interface{}) bool {
		return true
	})
	if err != nil {
		// Handle error
		return
	}
	for _, attackingEntity := range attackingEntities {
		attackingComponentUntyped, err := w.GetComponent(attackingEntity.ID, "AttackingComponent")
		if err != nil {
			// Handle error
			continue
		}
		attackingComponent, ok := attackingComponentUntyped.(*components.AttackingComponent)
		if !ok {
			// Handle type assertion error
			continue
		}
		if attackingComponent.TargetID != "" {
			// This entity has a target, so they try to attack it
			targetHealthComponentUntyped, err := w.GetComponent(attackingComponent.TargetID, "HealthComponent")
			if err != nil {
				// Handle error
				continue
			}
			targetHealth, ok := targetHealthComponentUntyped.(*components.HealthComponent)
			if ok {
				// The target has a HealthComponent, so it can be attacked
				targetHealth.TakeDamage(10)
				if playerComponentUntyped, err := w.GetComponent(attackingEntity.ID, "PlayerComponent"); err == nil {
					// The entity is a player, so we send them a message
					playerComponent, ok := playerComponentUntyped.(*components.PlayerComponent)
					if ok {
						playerComponent.Client.SendMessage(fmt.Sprintf("You attacked %s for 10 damage!", attackingComponent.TargetID))
					}
				}
			}
		}
	}
}
