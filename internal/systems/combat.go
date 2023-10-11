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

	attackingEntities, err := w.FindEntitiesByComponentPredicate("Combat", func(i interface{}) bool {
		return true
	})
	if err != nil {
		return
	}

	for _, attackingEntity := range attackingEntities {

		// .. A lot of things we need

		combatComponentUntyped, err := w.GetComponent(attackingEntity.ID, "Combat")
		if err != nil {
			log.Error().Msgf("Error getting attacking component: %v", err)
			return
		}
		combatComponent := combatComponentUntyped.(*components.Combat)
		if combatComponent.TargetID == "" {
			return
		}

		attackingPlayerUntyped, err := w.GetComponent(attackingEntity.ID, "Player")
		if err != nil {
			log.Error().Msgf("Error getting attacking player component: %v", err)
			return
		}
		attackingPlayer := attackingPlayerUntyped.(*components.Player)

		targetPlayerUntyped, err := w.GetComponent(combatComponent.TargetID, "Player")
		if err != nil {
			log.Error().Msgf("Error getting target player component: %v", err)
			return
		}
		targetPlayer := targetPlayerUntyped.(*components.Player)

		targetHealthUntyped, err := w.GetComponent(combatComponent.TargetID, "Health")
		if err != nil {
			log.Error().Msgf("Error getting target health component: %v", err)
			return
		}
		targetHealth := targetHealthUntyped.(*components.Health)

		// .. Are they dead yet?

		if targetHealth.CurrentHealth <= 0 {
			combatComponent.TargetID = ""

			w.RemoveComponent(attackingEntity.ID, "Combat")
			w.RemoveComponent(combatComponent.TargetID, "Combat")

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
