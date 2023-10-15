package systems

import (
	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/util"
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
		log.Error().Msgf("Error finding attacking entities: %v", err)
		return
	}

	for _, attackingEntity := range attackingEntities {
		combat, err := util.GetTypedComponent[components.Combat](w, attackingEntity.ID, "Combat")
		if err != nil {
			log.Error().Msgf("Error getting attacker combat component: %v", err)
			return
		}

		if combat.TargetID == "" {
			return
		}

		attackingPlayer, err := util.GetTypedComponent[components.Player](w, attackingEntity.ID, "Player")
		if err != nil {
			log.Error().Msgf("Error getting attacker player component: %v", err)
			return
		}

		targetPlayer, err := util.GetTypedComponent[components.Player](w, combat.TargetID, "Player")
		if err != nil {
			log.Error().Msgf("Error getting target player component: %v", err)
			return
		}

		targetHealth, err := util.GetTypedComponent[components.Health](w, combat.TargetID, "Health")
		if err != nil {
			log.Error().Msgf("Error getting target health component: %v", err)
			return
		}

		// .. Are they dead yet?

		if targetHealth.CurrentHealth <= 0 {
			combat.TargetID = ""

			w.RemoveComponent(attackingEntity.ID, "Combat")
			w.RemoveComponent(combat.TargetID, "Combat")

			attackingPlayer.Broadcast(fmt.Sprintf("You killed %s!", targetPlayer.Name))
			targetPlayer.Broadcast("You have died!")

			return
		}

		// .. Attack!

		damage := r.Intn(combat.MaxDamage-combat.MinDamage+1) + combat.MinDamage

		targetHealth.Lock()
		targetHealth.CurrentHealth -= damage
		targetHealth.Unlock()

		attackingPlayer.Broadcast(fmt.Sprintf("You attacked %s for %d damage!", targetPlayer.Name, damage))
		targetPlayer.Broadcast(fmt.Sprintf("%s attacked you for %d damage!", attackingPlayer.Name, damage))

		log.Trace().Msg(fmt.Sprintf("%s attacked %s for %d damage!", attackingPlayer.Name, targetPlayer.Name, damage))
	}
}
