package systems

import (
	"dmud/internal/components"
	"dmud/internal/ecs"
	"fmt"
	"math/rand"
	"time"

	"github.com/rs/zerolog/log"
)

type CombatSystem struct{}

func (cs *CombatSystem) Update(w *ecs.World, deltaTime float64) {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)

	attackingEntities, err := w.FindEntitiesByComponentPredicate("AttackingComponent", func(i interface{}) bool {
		return true
	})
	if err != nil {
		return
	}

	for _, attackingEntity := range attackingEntities {
		attackingComponentUntyped, err := w.GetComponent(attackingEntity.ID, "AttackingComponent")
		if err != nil {
			continue
		}

		attackingComponent := attackingComponentUntyped.(*components.AttackingComponent)
		if attackingComponent.TargetID == "" {
			continue
		}

		targetHealthComponentUntyped, err := w.GetComponent(attackingComponent.TargetID, "HealthComponent")
		if err != nil {
			continue
		}

		targetHealthComponent := targetHealthComponentUntyped.(*components.HealthComponent)
		if targetHealthComponent.CurrentHealth <= 0 {
			attackingComponent.TargetID = ""

			w.RemoveComponent(attackingEntity.ID, "AttackingComponent")
			w.RemoveComponent(attackingComponent.TargetID, "AttackingComponent")

			targetPlayerComponentUntyped, err := w.GetComponent(attackingComponent.TargetID, "PlayerComponent")
			if err != nil {
				continue
			}

			targetPlayerComponent := targetPlayerComponentUntyped.(*components.PlayerComponent)
			targetPlayerComponent.Broadcast("You have died!")

			attackingPlayerComponentUntyped, err := w.GetComponent(attackingEntity.ID, "PlayerComponent")
			if err != nil {
				continue
			}

			attackingPlayerComponent := attackingPlayerComponentUntyped.(*components.PlayerComponent)
			attackingPlayerComponent.Broadcast(fmt.Sprintf("You killed %s!", targetPlayerComponent.Name))

			continue
		}

		attackingComponent.Lock()
		damage := r.Intn(attackingComponent.MaxDamage-attackingComponent.MinDamage+1) + attackingComponent.MinDamage
		attackingComponent.Unlock()

		targetHealthComponent.Lock()
		targetHealthComponent.CurrentHealth -= damage
		targetHealthComponent.Unlock()

		attackerPlayerComponentUntyped, err := w.GetComponent(attackingEntity.ID, "PlayerComponent")
		if err != nil {
			continue
		}

		attackerPlayerComponent := attackerPlayerComponentUntyped.(*components.PlayerComponent)
		attackerPlayerComponent.Broadcast(fmt.Sprintf("You attacked %s for %d damage!", attackerPlayerComponent.Name, damage))

		log.Trace().Msg(fmt.Sprintf("%s attacked %s for %d damage!", attackerPlayerComponent.Name, attackingComponent.TargetID, damage))
	}
}
