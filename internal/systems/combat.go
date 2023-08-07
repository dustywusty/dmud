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

	attackingEntities, err := w.FindEntitiesByComponentPredicate("CombatComponent", func(i interface{}) bool {
		return true
	})
	if err != nil {
		return
	}

	for _, attackingEntity := range attackingEntities {

		// .. A lot of things we need

		combatComponentUntyped, err := w.GetComponent(attackingEntity.ID, "CombatComponent")
		if err != nil {
			log.Error().Msgf("Error getting attacking component: %v", err)
			return
		}
		combatComponent := combatComponentUntyped.(*components.CombatComponent)
		if combatComponent.TargetID == "" {
			return
		}

		attackingPlayerComponentUntyped, err := w.GetComponent(attackingEntity.ID, "PlayerComponent")
		if err != nil {
			log.Error().Msgf("Error getting attacking player component: %v", err)
			return
		}
		attackingPlayer := attackingPlayerComponentUntyped.(*components.PlayerComponent)

		targetPlayerComponentUntyped, err := w.GetComponent(combatComponent.TargetID, "PlayerComponent")
		if err != nil {
			log.Error().Msgf("Error getting target player component: %v", err)
			return
		}
		targetPlayer := targetPlayerComponentUntyped.(*components.PlayerComponent)

		targetHealthComponentUntyped, err := w.GetComponent(combatComponent.TargetID, "HealthComponent")
		if err != nil {
			log.Error().Msgf("Error getting target health component: %v", err)
			return
		}
		targetHealth := targetHealthComponentUntyped.(*components.HealthComponent)

		// .. Are they dead yet?

		if targetHealth.CurrentHealth <= 0 {
			combatComponent.TargetID = ""

			w.RemoveComponent(attackingEntity.ID, "CombatComponent")
			w.RemoveComponent(combatComponent.TargetID, "CombatComponent")

			targetPlayer.Broadcast("You have died!")
			attackingPlayer.Broadcast(fmt.Sprintf("You killed %s!", targetPlayer.Name))

			return
		}

		// .. Attack!

		damage := r.Intn(combatComponent.MaxDamage-combatComponent.MinDamage+1) + combatComponent.MinDamage

		targetHealth.Lock()
		targetHealth.CurrentHealth -= damage
		targetHealth.Unlock()

		attackingPlayer.Broadcast(fmt.Sprintf("You attacked %s for %d damage!", targetPlayer.Name, damage))
		targetPlayer.Broadcast(fmt.Sprintf("%s attacked you for %d damage!", attackingPlayer.Name, damage))

		log.Trace().Msg(fmt.Sprintf("%s attacked %s for %d damage!", attackingPlayer.Name, targetPlayer.Name, damage))
	}
}
