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

		// Broadcast state updates to players involved in combat
		if attackerPlayer != nil {
			broadcastStateToPlayer(w, attackingEntity.ID)
		}
		if targetPlayer != nil {
			broadcastStateToPlayer(w, targetID)
		}

		// Auto-retaliation: if target isn't already fighting back, make them attack the attacker
		targetCombat, err := getCombatComponent(w, targetID)
		if err != nil || targetCombat.TargetID == "" {
			// Target doesn't have combat component or isn't attacking anyone
			// Create or update combat component to attack back
			var minDamage, maxDamage int

			if targetPlayer != nil {
				// Default player damage if not specified
				minDamage = 10
				maxDamage = 50
			} else if targetNPC != nil {
				// Use NPC's damage from template
				if template, ok := components.NPCTemplates[targetNPC.TemplateID]; ok {
					minDamage = template.MinDamage
					maxDamage = template.MaxDamage
				} else {
					minDamage = 5
					maxDamage = 15
				}
			}

			retaliationCombat := &components.Combat{
				TargetID:  attackingEntity.ID,
				MinDamage: minDamage,
				MaxDamage: maxDamage,
			}

			// Find the target entity to add combat component
			targetEntities, _ := w.FindEntitiesByComponentPredicate("Health", func(i interface{}) bool {
				return true
			})
			for _, te := range targetEntities {
				if te.ID == targetID {
					w.AddComponent(&te, retaliationCombat)
					break
				}
			}
		}
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

		// Create player corpse
		spawnCorpse(w, targetPlayer.Name, targetID, true, targetPlayer.Area)

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

		// Create NPC corpse before removing entity
		spawnCorpse(w, targetNPC.Name, targetID, false, targetNPC.Area)

		// Remove NPC from world (spawn system will respawn it)
		w.RemoveEntity(targetID)
	}
}

// spawnCorpse creates a corpse entity at the location of death
func spawnCorpse(w components.WorldLike, victimName string, victimID common.EntityID, wasPlayer bool, area *components.Area) {
	if area == nil {
		return
	}

	// Create corpse entity
	corpseEntity := w.CreateEntity()

	// Create and add corpse component
	corpse := components.NewCorpse(victimName, victimID, wasPlayer, area)
	w.AddComponentToEntity(corpseEntity, corpse)

	log.Debug().Msgf("Spawned corpse of %s (entity: %s) at area (%d,%d,%d)",
		victimName, corpseEntity.GetID(), area.X, area.Y, area.Z)
}

func performAttack(attackerPlayer, targetPlayer *components.Player, attackerNPC, targetNPC *components.NPC,
	attackerName, targetName string, combat *components.Combat, targetHealth *components.Health) {

	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)

	damage := r.Intn(combat.MaxDamage-combat.MinDamage+1) + combat.MinDamage

	targetHealth.Lock()
	targetHealth.Current -= damage
	targetHealth.Unlock()

	// Send appropriate messages based on entity types
	if attackerPlayer != nil {
		attackerPlayer.Broadcast(fmt.Sprintf("You attacked %s for %d damage!", targetName, damage))
	}

	if targetPlayer != nil {
		targetPlayer.Broadcast(fmt.Sprintf("%s attacked you for %d damage!", attackerName, damage))
	}

	log.Trace().Msg(fmt.Sprintf("%s attacked %s for %d damage!", attackerName, targetName, damage))
}

func broadcastStateToPlayer(w *ecs.World, entityID common.EntityID) {
	player, err := ecs.GetTypedComponent[*components.Player](w, entityID, "Player")
	if err == nil && player != nil {
		player.BroadcastState(w.AsWorldLike(), entityID)
	}
}
