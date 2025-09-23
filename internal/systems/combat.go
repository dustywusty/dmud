package systems

import (
	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
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
			continue
		}

		if combat.TargetID == "" {
			continue
		}

		// Get attacker info (could be player or NPC)
		var attackerName string
		var attackerArea *components.Area
		attackerPlayer, _ := getPlayerComponent(w, attackingEntity.ID)
		attackerNPC, _ := getNPCComponent(w, attackingEntity.ID)

		if attackerPlayer != nil {
			attackerName = attackerPlayer.Name
			attackerArea = attackerPlayer.Area
		} else if attackerNPC != nil {
			attackerName = attackerNPC.Name
			attackerArea = attackerNPC.Area
		} else {
			// Neither player nor NPC
			w.RemoveComponent(attackingEntity.ID, "Combat")
			continue
		}

		// Get target info (could be player or NPC)
		targetID := common.EntityID(combat.TargetID)
		var targetName string
		var targetArea *components.Area
		targetPlayer, _ := getPlayerComponent(w, targetID)
		targetNPC, _ := getNPCComponent(w, targetID)

		if targetPlayer != nil {
			targetName = targetPlayer.Name
			targetArea = targetPlayer.Area
		} else if targetNPC != nil {
			targetName = targetNPC.Name
			targetArea = targetNPC.Area
		} else {
			// Target no longer exists
			combat.TargetID = ""
			if attackerPlayer != nil {
				attackerPlayer.Broadcast("Your target is no longer valid.")
			}
			continue
		}

		// Verify both are in same area
		if attackerArea != targetArea {
			combat.TargetID = ""
			if attackerPlayer != nil {
				attackerPlayer.Broadcast("Your target is no longer here.")
			}
			continue
		}

		targetHealth, err := getHealthComponent(w, targetID)
		if err != nil {
			combat.TargetID = ""
			if attackerPlayer != nil {
				attackerPlayer.Broadcast("Your target cannot be damaged.")
			}
			continue
		}

		if isTargetDead(targetHealth) {
			handleTargetDeath(w.AsWorldLike(), attackingEntity.ID, targetID,
				attackerPlayer, targetPlayer, attackerNPC, targetNPC)
			continue
		}

		performAttack(attackerPlayer, targetPlayer, attackerNPC, targetNPC,
			attackerName, targetName, combat, targetHealth)
	}
}

func findAttackingEntities(w *ecs.World) ([]ecs.Entity, error) {
	return w.FindEntitiesByComponentPredicate("Combat", func(i interface{}) bool {
		c := i.(*components.Combat)
		return c.TargetID != ""
	})
}

func getCombatComponent(w *ecs.World, entityID common.EntityID) (*components.Combat, error) {
	return ecs.GetTypedComponent[*components.Combat](w, entityID, "Combat")
}

func getPlayerComponent(w *ecs.World, entityID common.EntityID) (*components.Player, error) {
	return ecs.GetTypedComponent[*components.Player](w, entityID, "Player")
}

func getNPCComponent(w *ecs.World, entityID common.EntityID) (*components.NPC, error) {
	return ecs.GetTypedComponent[*components.NPC](w, entityID, "NPC")
}

func getHealthComponent(w *ecs.World, entityID common.EntityID) (*components.Health, error) {
	return ecs.GetTypedComponent[*components.Health](w, entityID, "Health")
}

func isTargetDead(health *components.Health) bool {
	return health.Current <= 0
}

func handleTargetDeath(w components.WorldLike, attackerID common.EntityID, targetID common.EntityID,
	attackerPlayer, targetPlayer *components.Player, attackerNPC, targetNPC *components.NPC) {

	// Clear combat states
	if combatComp, err := w.GetComponent(attackerID, "Combat"); err == nil {
		combat := combatComp.(*components.Combat)
		combat.TargetID = ""
	}
	if combatComp, err := w.GetComponent(targetID, "Combat"); err == nil {
		combat := combatComp.(*components.Combat)
		combat.TargetID = ""
	}

	// Handle different death scenarios
	if targetPlayer != nil {
		// Player died
		targetPlayer.Broadcast("You have died!")
		if attackerPlayer != nil {
			attackerPlayer.Broadcast(fmt.Sprintf("You killed %s!", targetPlayer.Name))
			targetPlayer.Area.Broadcast(fmt.Sprintf("%s has been slain by %s!", targetPlayer.Name, attackerPlayer.Name))
		} else if attackerNPC != nil {
			targetPlayer.Area.Broadcast(fmt.Sprintf("%s has been slain by %s!", targetPlayer.Name, attackerNPC.Name))
		}

		// TODO: Handle respawn
		// For now, restore health
		if healthComp, err := w.GetComponent(targetID, "Health"); err == nil {
			health := healthComp.(*components.Health)
			health.Current = health.Max
			// health.UpdateStatus()
			targetPlayer.Broadcast("You have been revived with full health.")
		}
	} else if targetNPC != nil {
		// NPC died
		if targetNPC.Area != nil {
			targetNPC.Area.Broadcast(targetNPC.Name + " has been slain!")
		}

		if attackerPlayer != nil {
			attackerPlayer.Broadcast("You have defeated " + targetNPC.Name + "!")
			// TODO: Award experience/loot
		}

		// Remove NPC from world (spawn system will respawn it)
		w.RemoveEntity(targetID)
	}
}

func performAttack(attackerPlayer, targetPlayer *components.Player, attackerNPC, targetNPC *components.NPC,
	attackerName, targetName string, combat *components.Combat, targetHealth *components.Health) {

	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)

	damage := r.Intn(combat.MaxDamage-combat.MinDamage+1) + combat.MinDamage

	targetHealth.Lock()
	defer targetHealth.Unlock()
	targetHealth.Current -= damage

	// Send appropriate messages based on entity types
	if attackerPlayer != nil {
		attackerPlayer.Broadcast(fmt.Sprintf("You attacked %s for %d damage!", targetName, damage))
	}

	if targetPlayer != nil {
		targetPlayer.Broadcast(fmt.Sprintf("%s attacked you for %d damage!", attackerName, damage))
	}

	log.Trace().Msg(fmt.Sprintf("%s attacked %s for %d damage!", attackerName, targetName, damage))
}
