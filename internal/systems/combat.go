package systems

import (
	"dmud/internal/common"
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
	attackingEntities, err := findAttackingEntities(w)
	if err != nil {
		log.Error().Msgf("Error finding attacking entities: %v", err)
		return
	}

	for _, attackingEntity := range attackingEntities {
		combat, err := getCombatComponent(w, attackingEntity.ID)
		if err != nil {
			log.Error().Msgf("Error getting attacker combat component: %v", err)
			return
		}

		if combat.TargetID == "" {
			return
		}

		attackingPlayer, err := getPlayerComponent(w, attackingEntity.ID)
		if err != nil {
			log.Error().Msgf("Error getting attacker player component: %v", err)
			return
		}

		targetPlayer, err := getPlayerComponent(w, combat.TargetID)
		if err != nil {
			log.Error().Msgf("Error getting target player component: %v", err)
			return
		}

		targetHealth, err := getHealthComponent(w, combat.TargetID)
		if err != nil {
			log.Error().Msgf("Error getting target health component: %v", err)
			return
		}

		if isTargetDead(targetHealth) {
			handleTargetDeath(w, attackingEntity.ID, combat.TargetID, attackingPlayer, targetPlayer)
			return
		}

		attackPlayer(attackingPlayer, targetPlayer, combat, targetHealth)
	}
}

func findAttackingEntities(w *ecs.World) ([]ecs.Entity, error) {
	return w.FindEntitiesByComponentPredicate("Combat", func(i interface{}) bool {
		return true
	})
}

func getCombatComponent(w *ecs.World, entityID common.EntityID) (*components.Combat, error) {
	return util.GetTypedComponent[*components.Combat](w, entityID, "Combat")
}

func getPlayerComponent(w *ecs.World, entityID common.EntityID) (*components.Player, error) {
	return util.GetTypedComponent[*components.Player](w, entityID, "Player")
}

func getHealthComponent(w *ecs.World, entityID common.EntityID) (*components.Health, error) {
	return util.GetTypedComponent[*components.Health](w, entityID, "Health")
}

func isTargetDead(health *components.Health) bool {
	return health.Current <= 0
}

func handleTargetDeath(w *ecs.World, attackerID common.EntityID, targetID common.EntityID, attackerPlayer, targetPlayer *components.Player) {
	combat := &components.Combat{}
	combat.TargetID = ""

	w.RemoveComponent(attackerID, "Combat")
	w.RemoveComponent(targetID, "Combat")

	attackerPlayer.Broadcast(fmt.Sprintf("You killed %s!", targetPlayer.Name))
	targetPlayer.Broadcast("You have died!")
}

func attackPlayer(attackerPlayer, targetPlayer *components.Player, combat *components.Combat, targetHealth *components.Health) {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)

	damage := r.Intn(combat.MaxDamage-combat.MinDamage+1) + combat.MinDamage

	targetHealth.Lock()
	targetHealth.Current -= damage
	targetHealth.Unlock()

	attackerPlayer.Broadcast(fmt.Sprintf("You attacked %s for %d damage!", targetPlayer.Name, damage))
	targetPlayer.Broadcast(fmt.Sprintf("%s attacked you for %d damage!", attackerPlayer.Name, damage))

	log.Trace().Msg(fmt.Sprintf("%s attacked %s for %d damage!", attackerPlayer.Name, targetPlayer.Name, damage))
}
