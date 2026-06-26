package systems

import (
	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/util"
	"encoding/json"
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

		if attackerNPC != nil && hasSuppressedAggro(w, attackingEntity.ID) {
			combat.TargetID = ""
			combat.TargetQueue = nil
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

		performAttack(w, attackingEntity.ID, attackerPlayer, targetPlayer, attackerNPC, targetNPC,
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
		if targetNPC != nil && hasSuppressedRetaliation(w, targetID) {
			if err == nil {
				targetCombat.TargetID = ""
				targetCombat.TargetQueue = nil
			}
			continue
		}
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

func hasSuppressedAggro(w *ecs.World, entityID common.EntityID) bool {
	statusEffects, err := ecs.GetTypedComponent[*components.StatusEffects](w, entityID, "StatusEffects")
	if err != nil || statusEffects == nil {
		return false
	}
	return statusEffects.HasSuppressedAggro()
}

func hasSuppressedRetaliation(w *ecs.World, entityID common.EntityID) bool {
	statusEffects, err := ecs.GetTypedComponent[*components.StatusEffects](w, entityID, "StatusEffects")
	if err != nil || statusEffects == nil {
		return false
	}
	return statusEffects.HasSuppressedRetaliation()
}

func isTargetDead(health *components.Health) bool {
	return health.Current <= 0
}

func handleTargetDeath(w components.WorldLike, attackerID common.EntityID, targetID common.EntityID,
	attackerPlayer, targetPlayer *components.Player, attackerNPC, targetNPC *components.NPC) {

	// Clear or switch to next target in queue
	if combatComp, err := w.GetComponent(attackerID, "Combat"); err == nil {
		combat := combatComp.(*components.Combat)
		if len(combat.TargetQueue) > 0 {
			// Switch to next target in queue
			combat.TargetID = combat.TargetQueue[0]
			combat.TargetQueue = combat.TargetQueue[1:]

			// Announce switching targets if it's a player
			if attackerPlayer != nil && targetNPC != nil {
				// Find the new target's name
				newTargetID := combat.TargetID
				if newTargetNPC, err := w.GetComponent(newTargetID, "NPC"); err == nil {
					newNPC := newTargetNPC.(*components.NPC)
					attackerPlayer.Broadcast(util.TagMessage("DMG", fmt.Sprintf("You turn your attention to %s!", newNPC.Name)))
				}
			}
		} else {
			combat.TargetID = ""
		}
	}
	if combatComp, err := w.GetComponent(targetID, "Combat"); err == nil {
		combat := combatComp.(*components.Combat)
		combat.TargetID = ""
	}

	// Handle different death scenarios
	if targetPlayer != nil {
		// Player died
		targetPlayer.Broadcast(util.TagMessageWithStatus("DMG", "DEATH", "You have died!"))
		if attackerPlayer != nil {
			attackerPlayer.Broadcast(util.TagMessage("DMG", fmt.Sprintf("You killed %s!", targetPlayer.Name)))
			targetPlayer.Area.Broadcast(util.TagMessageWithStatus("DMG", "DEATH", fmt.Sprintf("%s has been slain by %s!", targetPlayer.Name, attackerPlayer.Name)))
		} else if attackerNPC != nil {
			targetPlayer.Area.Broadcast(util.TagMessageWithStatus("DMG", "DEATH", fmt.Sprintf("%s has been slain by %s!", targetPlayer.Name, attackerNPC.Name)))
		}

		// Create player corpse with their inventory (worn gear drops too)
		var corpseInventory *components.Inventory
		if invComp, err := w.GetComponent(targetID, "Inventory"); err == nil {
			corpseInventory = invComp.(*components.Inventory)
		}
		dropEquipmentInto(w, targetID, corpseInventory)
		spawnCorpse(w, targetPlayer.Name, targetID, true, targetPlayer.Area, corpseInventory)

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
			targetNPC.Area.Broadcast(util.TagMessageWithStatus("DMG", "DEATH", targetNPC.Name+" has been slain!"))
		}

		if attackerPlayer != nil {
			attackerPlayer.Broadcast(util.TagMessage("DMG", "You have defeated "+targetNPC.Name+"!"))

			// Coin drops, scaled like XP off the NPC's threat.
			goldReward := 1 + rand.Intn(3)
			if template, ok := components.NPCTemplates[targetNPC.TemplateID]; ok && template.MaxDamage > 0 {
				goldReward = template.MaxDamage*2 + rand.Intn(template.MaxDamage+1)
			}
			if goldReward < 1 {
				goldReward = 1
			}
			if invComp, err := w.GetComponent(attackerID, "Inventory"); err == nil {
				if inv, ok := invComp.(*components.Inventory); ok {
					inv.AddItem(components.CreateItem("gold_coin", goldReward))
					attackerPlayer.Broadcast(util.TagMessage("STATUS", fmt.Sprintf("You loot %d gold from %s.", goldReward, targetNPC.Name)))
				}
			}

			// Award experience based on NPC level/difficulty
			xpReward := 50
			if template, ok := components.NPCTemplates[targetNPC.TemplateID]; ok {
				xpReward = template.MaxDamage * 5
			}

			if expComp, err := w.GetComponent(attackerID, "Experience"); err == nil {
				experience := expComp.(*components.Experience)
				leveledUp, newLevel := experience.AddXP(xpReward)

				attackerPlayer.Broadcast(util.TagMessage("STATUS", fmt.Sprintf("You gained %d experience!", xpReward)))

				if leveledUp {
					attackerPlayer.Broadcast(util.TagMessage("STATUS", fmt.Sprintf("You have reached level %d!", newLevel)))

					// Scale up player health on level up and heal to full
					if healthComp, err := w.GetComponent(attackerID, "Health"); err == nil {
						health := healthComp.(*components.Health)
						oldMax := health.Max
						newMax := int(float64(100) * components.GetLevelScaling(newLevel))
						hpGain := newMax - oldMax
						health.Max = newMax
						health.Current = newMax
						attackerPlayer.Broadcast(fmt.Sprintf("Your maximum health increased by %d and you are fully healed!", hpGain))
					}
				}

				// Broadcast state update to show XP and possibly level change
				attackerPlayer.BroadcastState(w, attackerID)
			}
		}

		// Create NPC corpse with their inventory before removing entity (gear drops)
		var corpseInventory *components.Inventory
		if invComp, err := w.GetComponent(targetID, "Inventory"); err == nil {
			corpseInventory = invComp.(*components.Inventory)
		}
		dropEquipmentInto(w, targetID, corpseInventory)
		spawnCorpse(w, targetNPC.Name, targetID, false, targetNPC.Area, corpseInventory)

		// Remove NPC from world (spawn system will respawn it)
		w.RemoveEntity(targetID)
	}
}

// dropEquipmentInto moves a dying creature's worn gear into the corpse inventory
// so it can be looted, just like carried items.
func dropEquipmentInto(w components.WorldLike, entityID common.EntityID, inv *components.Inventory) {
	if inv == nil {
		return
	}
	eqComp, err := w.GetComponent(entityID, "Equipment")
	if err != nil {
		return
	}
	eq, ok := eqComp.(*components.Equipment)
	if !ok {
		return
	}
	for _, slot := range []components.EquipSlot{components.SlotWeapon, components.SlotArmor, components.SlotShield} {
		if it := eq.Remove(slot); it != nil {
			inv.AddItem(it)
		}
	}
}

// spawnCorpse creates a corpse entity at the location of death
func spawnCorpse(w components.WorldLike, victimName string, victimID common.EntityID, wasPlayer bool, area *components.Area, inventory *components.Inventory) {
	if area == nil {
		return
	}

	// Create corpse entity
	corpseEntity := w.CreateEntity()

	// Create and add corpse component with inventory
	corpse := components.NewCorpse(victimName, victimID, wasPlayer, area, inventory)
	w.AddComponentToEntity(corpseEntity, corpse)
	if area != nil {
		area.MarkDirty()
	}

	log.Debug().Msgf("Spawned corpse of %s (entity: %s) at area (%d,%d,%d)",
		victimName, corpseEntity.GetID(), area.X, area.Y, area.Z)
}

func performAttack(w *ecs.World, attackerID common.EntityID, attackerPlayer, targetPlayer *components.Player, attackerNPC, targetNPC *components.NPC,
	attackerName, targetName string, combat *components.Combat, targetHealth *components.Health) {

	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)

	baseDamage := r.Intn(combat.MaxDamage-combat.MinDamage+1) + combat.MinDamage
	damage := baseDamage

	if attackerPlayer != nil {
		experience, _ := ecs.GetTypedComponent[*components.Experience](w, attackerID, "Experience")
		if experience != nil {
			level := experience.GetLevel()
			scaling := components.GetLevelScaling(level)
			damage = int(float64(baseDamage) * scaling)
		}
		if stats, err := ecs.GetTypedComponent[*components.Stats](w, attackerID, "Stats"); err == nil && stats != nil {
			damage = int(float64(damage) * stats.MeleeFactor()) // Strength hits harder
		}
	}

	// NPCs are statted too: a brawny ogre hits harder than its base damage.
	if attackerNPC != nil {
		if stats, err := ecs.GetTypedComponent[*components.Stats](w, attackerID, "Stats"); err == nil && stats != nil {
			damage = int(float64(damage) * stats.MeleeFactor())
		}
	}

	// Equipped weapon adds to the swing; the target's worn armor soaks part of it.
	if eq, e := ecs.GetTypedComponent[*components.Equipment](w, attackerID, "Equipment"); e == nil && eq != nil {
		if lo, hi := eq.WeaponDamage(); hi > 0 && hi >= lo {
			damage += lo + r.Intn(hi-lo+1)
		}
	}
	if eq, e := ecs.GetTypedComponent[*components.Equipment](w, combat.TargetID, "Equipment"); e == nil && eq != nil {
		if armor := eq.ArmorValue(); armor > 0 {
			if damage -= armor; damage < 1 {
				damage = 1
			}
		}
	}

	targetHealth.Current -= damage

	// Send appropriate messages based on entity types
	if attackerPlayer != nil {
		attackerPlayer.Broadcast(util.TagMessage("DMG", fmt.Sprintf("You attacked %s for %d damage!", targetName, damage)))
		emitCombatEvent(attackerPlayer, targetName, damage, targetHealth)
		if stats, err := ecs.GetTypedComponent[*components.Stats](w, attackerID, "Stats"); err == nil {
			components.TrainStat(attackerPlayer, stats, components.STR) // swinging trains Strength
		}
	}

	if targetPlayer != nil {
		targetPlayer.Broadcast(util.TagMessage("DMG", fmt.Sprintf("%s attacked you for %d damage!", attackerName, damage)))
		if stats, err := ecs.GetTypedComponent[*components.Stats](w, combat.TargetID, "Stats"); err == nil {
			components.TrainStat(targetPlayer, stats, components.CON) // taking hits trains Constitution
		}
	}

	log.Trace().Msg(fmt.Sprintf("%s attacked %s for %d damage!", attackerName, targetName, damage))
}

// emitCombatEvent sends the structured combat event to a player attacker so the
// client can draw the target's HP bar. Shared by melee and spell damage.
func emitCombatEvent(attacker *components.Player, targetName string, damage int, targetHealth *components.Health) {
	if attacker == nil {
		return
	}
	hp := targetHealth.Current
	if hp < 0 {
		hp = 0
	}
	ev, _ := json.Marshal(struct {
		Type      string `json:"type"`
		Target    string `json:"target"`
		Damage    int    `json:"damage"`
		TargetHP  int    `json:"target_hp"`
		TargetMax int    `json:"target_max"`
		Killed    bool   `json:"killed"`
	}{"combat", targetName, damage, hp, targetHealth.Max, targetHealth.Current <= 0})
	attacker.Broadcast(util.TagMessage("EVENT", string(ev)))
}

// ApplyPlayerSpellDamage deals direct (non-melee) damage from a player caster to
// a target NPC: it lowers the target's health, emits the combat event so the
// client's enemy HP bar updates, and on a lethal hit runs the normal death flow
// (XP, level-up, corpse, removal). Returns the damage dealt and whether the
// target was slain. It is a no-op (0, false) if the target lacks Health/NPC.
func ApplyPlayerSpellDamage(w *ecs.World, casterID, targetID common.EntityID, amount int) (int, bool) {
	targetNPC, err := getNPCComponent(w, targetID)
	if err != nil || targetNPC == nil {
		return 0, false
	}
	targetHealth, err := getHealthComponent(w, targetID)
	if err != nil {
		return 0, false
	}
	if amount < 0 {
		amount = 0
	}

	caster, _ := getPlayerComponent(w, casterID)

	targetHealth.Current -= amount
	killed := targetHealth.Current <= 0

	emitCombatEvent(caster, targetNPC.Name, amount, targetHealth)

	if killed {
		handleTargetDeath(w.AsWorldLike(), casterID, targetID, caster, nil, nil, targetNPC)
	}
	return amount, killed
}

func broadcastStateToPlayer(w *ecs.World, entityID common.EntityID) {
	player, err := ecs.GetTypedComponent[*components.Player](w, entityID, "Player")
	if err == nil && player != nil {
		player.BroadcastState(w.AsWorldLike(), entityID)
	}
}
