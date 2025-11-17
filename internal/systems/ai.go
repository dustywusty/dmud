package systems

import (
	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
	"fmt"
	"math/rand"
	"time"
)

const wanderMinimumInterval = 20 * time.Second

type AISystem struct {
	lastUpdate time.Time
}

func NewAISystem() *AISystem {
	return &AISystem{
		lastUpdate: time.Now(),
	}
}

func (as *AISystem) Update(w *ecs.World, deltaTime float64) {
	if time.Since(as.lastUpdate) < 2*time.Second {
		return
	}
	as.lastUpdate = time.Now()

	npcEntities, err := w.FindEntitiesByComponentPredicate("NPC", func(i interface{}) bool {
		return true
	})
	if err != nil {
		return
	}

	for _, entity := range npcEntities {
		as.processNPCBehavior(w, entity)
	}
}

func (as *AISystem) processNPCBehavior(w *ecs.World, npcEntity ecs.Entity) {
	npc, err := ecs.GetTypedComponent[*components.NPC](w, npcEntity.ID, "NPC")
	if err != nil {
		return
	}

	health, err := ecs.GetTypedComponent[*components.Health](w, npcEntity.ID, "Health")
	if err != nil {
		return
	}

	if health.Status == components.Dead {
		return
	}

	combat, _ := ecs.GetTypedComponent[*components.Combat](w, npcEntity.ID, "Combat")

	as.attemptWander(w, npcEntity, npc, combat)

	switch npc.Behavior {
	case components.BehaviorAggressive:
		as.processAggressiveNPC(w, npcEntity, npc, combat)
	case components.BehaviorFriendly, components.BehaviorMerchant:
		as.processFriendlyNPC(w, npcEntity, npc)
	case components.BehaviorGuard:
		as.processGuardNPC(w, npcEntity, npc)
	case components.BehaviorPassive:
		as.processPassiveNPC(w, npcEntity, npc)
	}
}

func (as *AISystem) processAggressiveNPC(w *ecs.World, npcEntity ecs.Entity, npc *components.NPC, combat *components.Combat) {
	if combat == nil {
		return
	}

	combat.RLock()
	hasTarget := combat.TargetID != ""
	minDamage := combat.MinDamage
	maxDamage := combat.MaxDamage
	combat.RUnlock()

	// If not in combat, look for targets
	if !hasTarget {
		npc.RLock()
		area := npc.Area
		npc.RUnlock()

		if area != nil && len(area.Players) > 0 {
			// Pick a random player to attack
			area.PlayersMutex.RLock()
			if len(area.Players) > 0 {
				target := area.Players[rand.Intn(len(area.Players))]
				area.PlayersMutex.RUnlock()

				// Find player entity
				playerEntities, _ := w.FindEntitiesByComponentPredicate("Player", func(i interface{}) bool {
					p, ok := i.(*components.Player)
					return ok && p == target
				})

				if len(playerEntities) > 0 {
					// Start combat
					newCombat := &components.Combat{
						TargetID:  playerEntities[0].ID,
						MinDamage: minDamage,
						MaxDamage: maxDamage,
					}
					w.AddComponent(&npcEntity, newCombat)

					area.Broadcast(npc.Name + " attacks " + target.Name + "!")
				}
			} else {
				area.PlayersMutex.RUnlock()
			}
		}
	}
}

func (as *AISystem) attemptWander(_ *ecs.World, _ ecs.Entity, npc *components.NPC, combat *components.Combat) {
	if combat != nil {
		combat.RLock()
		inCombat := combat.TargetID != ""
		combat.RUnlock()
		if inCombat {
			return
		}
	}

	npc.RLock()
	currentArea := npc.Area
	lastMovement := npc.LastMovement
	name := npc.Name
	templateID := npc.TemplateID
	npc.RUnlock()

	// Check if this NPC is stationary
	if template, ok := components.NPCTemplates[templateID]; ok && template.Stationary {
		return
	}

	if currentArea == nil {
		return
	}

	if time.Since(lastMovement) < wanderMinimumInterval {
		return
	}

	if rand.Float64() > 0.5 {
		return
	}

	exits := regionExits(currentArea)
	if len(exits) == 0 {
		return
	}

	chosenExit := exits[rand.Intn(len(exits))]
	destination := chosenExit.Area
	if destination == nil || destination == currentArea {
		return
	}

	currentArea.Broadcast(name + " wanders " + chosenExit.Direction + ".")

	npc.Lock()
	if npc.Area != currentArea {
		npc.Unlock()
		return
	}
	npc.Area = destination
	npc.LastMovement = time.Now()
	npc.Unlock()

	destination.Broadcast(name + " wanders in.")
}

func regionExits(area *components.Area) []components.Exit {
	if area == nil {
		return nil
	}

	region := area.Region
	exits := make([]components.Exit, 0, len(area.Exits))
	for _, exit := range area.Exits {
		if exit.Area == nil {
			continue
		}
		if exit.Area.Region != region {
			continue
		}
		exits = append(exits, exit)
	}
	return exits
}

func (as *AISystem) processFriendlyNPC(_ *ecs.World, npcEntity ecs.Entity, npc *components.NPC) {
	// Occasionally say something
	if time.Since(npc.LastAction) > 30*time.Second && rand.Float64() < 0.3 {
		dialogue := npc.GetRandomDialogue()
		if dialogue != "" && npc.Area != nil {
			npc.Area.Broadcast(npc.Name + " says: " + dialogue)
			npc.Lock()
			defer npc.Unlock()
			npc.LastAction = time.Now()
		}
	}
}

func (as *AISystem) processGuardNPC(w *ecs.World, npcEntity ecs.Entity, npc *components.NPC) {
	combat, _ := ecs.GetTypedComponent[*components.Combat](w, npcEntity.ID, "Combat")
	if as.guardIntervene(w, npcEntity, npc, combat) {
		return
	}

	as.guardBlessPlayers(w, npc)
	as.processFriendlyNPC(w, npcEntity, npc)
}

func (as *AISystem) guardBlessPlayers(w *ecs.World, npc *components.NPC) {
	npc.RLock()
	area := npc.Area
	lastAction := npc.LastAction
	npc.RUnlock()

	if area == nil {
		return
	}

	if time.Since(lastAction) < 10*time.Second {
		return
	}

	area.PlayersMutex.RLock()
	players := make([]*components.Player, len(area.Players))
	copy(players, area.Players)
	area.PlayersMutex.RUnlock()

	for _, player := range players {
		playerEntities, _ := w.FindEntitiesByComponentPredicate("Player", func(i interface{}) bool {
			p, ok := i.(*components.Player)
			return ok && p == player
		})

		if len(playerEntities) == 0 {
			continue
		}

		playerEntity := playerEntities[0]

		health, err := ecs.GetTypedComponent[*components.Health](w, playerEntity.ID, "Health")
		if err != nil {
			continue
		}

		statusEffects, err := ecs.GetTypedComponent[*components.StatusEffects](w, playerEntity.ID, "StatusEffects")
		if err != nil || statusEffects == nil {
			statusEffects = components.NewStatusEffects()
			w.AddComponent(&playerEntity, statusEffects)
		}

		hasBlessing := statusEffects.HasEffect(components.StatusEffectGuardBlessing)
		isInjured := health.Current < health.Max

		// Heal and bless travelers who don't have the blessing
		if !hasBlessing {
			effect := components.StatusEffect{
				Type:      components.StatusEffectGuardBlessing,
				Name:      "Guard's Blessing",
				AppliedAt: time.Now(),
				Duration:  10 * time.Minute,
				HPBonus:   500,
				Applied:   false,
			}

			statusEffects.AddEffect(effect)

			blessings := []string{
				"May the light protect you, traveler.",
				"The Guard grants you their blessing.",
				"You are under the Guard's protection.",
				"Stay safe in these lands, friend.",
			}
			message := blessings[rand.Intn(len(blessings))]

			area.Broadcast(fmt.Sprintf("%s says: \"%s\"", npc.Name, message))

			// Heal to full health first
			health.Lock()
			wasInjured := health.Current < health.Max
			health.Current = health.Max
			// Then add the buff HP
			health.Current += 500
			effectiveMax := health.GetEffectiveMax(500)
			if health.Current > effectiveMax {
				health.Current = effectiveMax
			}
			health.Unlock()

			if wasInjured {
				player.Broadcast("The guard's healing power restores you to full health!")
			}
			player.Broadcast("You feel a surge of vitality as the guard blesses you! (+500 HP)")

			// Broadcast state update to player
			player.BroadcastState(w.AsWorldLike(), playerEntity.ID)

			npc.Lock()
			npc.LastAction = time.Now()
			npc.Unlock()

			return
		}

		// If they have the blessing but are injured, just heal them
		if hasBlessing && isInjured {
			health.Lock()
			hpBonus := statusEffects.GetTotalHPBonus()
			effectiveMax := health.Max + hpBonus
			health.Current = effectiveMax
			health.Unlock()

			healings := []string{
				"Rest easy, traveler. Your wounds are healed.",
				"The Guard's light mends your injuries.",
				"Be whole again, friend.",
			}
			message := healings[rand.Intn(len(healings))]

			area.Broadcast(fmt.Sprintf("%s says: \"%s\"", npc.Name, message))
			player.Broadcast("The guard heals your wounds completely!")

			// Broadcast state update to player
			player.BroadcastState(w.AsWorldLike(), playerEntity.ID)

			npc.Lock()
			npc.LastAction = time.Now()
			npc.Unlock()

			return
		}
	}
}

func (as *AISystem) guardIntervene(w *ecs.World, npcEntity ecs.Entity, npc *components.NPC, combat *components.Combat) bool {
	npc.RLock()
	area := npc.Area
	npc.RUnlock()

	if area == nil {
		return false
	}

	if combat != nil {
		combat.RLock()
		hasTarget := combat.TargetID != ""
		combat.RUnlock()

		if hasTarget {
			return true
		}
	}

	aggressorID, aggressorName := findAreaAggressor(w, area, npcEntity.ID)
	if aggressorID == "" {
		return false
	}

	if combat == nil {
		minDamage := 5
		maxDamage := 10
		if template, ok := components.NPCTemplates[npc.TemplateID]; ok {
			minDamage = template.MinDamage
			maxDamage = template.MaxDamage
		}

		combat = &components.Combat{
			MinDamage: minDamage,
			MaxDamage: maxDamage,
		}
		w.AddComponent(&npcEntity, combat)
	}

	combat.Lock()
	combat.TargetID = aggressorID
	combat.Unlock()

	area.Broadcast(fmt.Sprintf("%s shouts, \"Keep the peace!\" and attacks %s!", npc.Name, aggressorName))

	return true
}

func findAreaAggressor(w *ecs.World, area *components.Area, guardID common.EntityID) (common.EntityID, string) {
	if area == nil {
		return "", ""
	}

	npcEntities, _ := w.FindEntitiesByComponentPredicate("NPC", func(i interface{}) bool {
		npc, ok := i.(*components.NPC)
		if !ok {
			return false
		}
		return npc.Area == area
	})

	for _, entity := range npcEntities {
		if entity.ID == guardID {
			continue
		}

		npcComp, err := ecs.GetTypedComponent[*components.NPC](w, entity.ID, "NPC")
		if err != nil {
			continue
		}

		if npcComp.Behavior == components.BehaviorGuard {
			continue
		}

		targetID, targetPlayer, targetNPC := getCombatTargetInfo(w, entity.ID)
		if !guardShouldIntervene(area, guardID, targetID, targetPlayer, targetNPC) {
			continue
		}

		return entity.ID, npcComp.Name
	}

	playerEntities, _ := w.FindEntitiesByComponentPredicate("Player", func(i interface{}) bool {
		player, ok := i.(*components.Player)
		if !ok {
			return false
		}
		return player.Area == area
	})

	for _, entity := range playerEntities {
		if entity.ID == guardID {
			continue
		}

		playerComp, err := ecs.GetTypedComponent[*components.Player](w, entity.ID, "Player")
		if err != nil {
			continue
		}

		targetID, targetPlayer, targetNPC := getCombatTargetInfo(w, entity.ID)
		if !guardShouldIntervene(area, guardID, targetID, targetPlayer, targetNPC) {
			continue
		}

		return entity.ID, playerComp.Name
	}

	return "", ""
}

func getCombatTargetInfo(w *ecs.World, entityID common.EntityID) (common.EntityID, *components.Player, *components.NPC) {
	combatComp, err := ecs.GetTypedComponent[*components.Combat](w, entityID, "Combat")
	if err != nil {
		return "", nil, nil
	}

	combatComp.RLock()
	targetID := combatComp.TargetID
	combatComp.RUnlock()

	if targetID == "" {
		return "", nil, nil
	}

	targetPlayer, _ := ecs.GetTypedComponent[*components.Player](w, targetID, "Player")
	targetNPC, _ := ecs.GetTypedComponent[*components.NPC](w, targetID, "NPC")

	return targetID, targetPlayer, targetNPC
}

func guardShouldIntervene(area *components.Area, guardID, targetID common.EntityID, targetPlayer *components.Player, targetNPC *components.NPC) bool {
	if targetID == "" {
		return false
	}

	if targetID == guardID {
		return true
	}

	if targetPlayer != nil && targetPlayer.Area == area {
		return true
	}

	if targetNPC != nil && targetNPC.Area == area {
		if targetNPC.Behavior != components.BehaviorAggressive && targetNPC.Behavior != components.BehaviorGuard {
			return true
		}
	}

	return false
}

func (as *AISystem) processPassiveNPC(w *ecs.World, npcEntity ecs.Entity, npc *components.NPC) {
	// Passive NPCs might flee when attacked or just emote
	if time.Since(npc.LastAction) > 45*time.Second && rand.Float64() < 0.2 {
		dialogue := npc.GetRandomDialogue()
		if dialogue != "" && npc.Area != nil {
			npc.Area.Broadcast(npc.Name + " " + dialogue)
			npc.Lock()
			defer npc.Unlock()
			npc.LastAction = time.Now()
		}
	}
}
