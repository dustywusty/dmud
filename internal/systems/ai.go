// internal/systems/ai.go
package systems

import (
	"dmud/internal/components"
	"dmud/internal/ecs"
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
	// Only update AI every 2 seconds
	if time.Since(as.lastUpdate) < 2*time.Second {
		return
	}
	as.lastUpdate = time.Now()

	// Find all NPCs
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

	// Don't process dead NPCs
	if health.Status == components.Dead {
		return
	}

	combat, _ := ecs.GetTypedComponent[*components.Combat](w, npcEntity.ID, "Combat")

	as.attemptWander(w, npcEntity, npc, combat)

	// Process based on behavior type
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

func (as *AISystem) attemptWander(w *ecs.World, npcEntity ecs.Entity, npc *components.NPC, combat *components.Combat) {
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
	npc.RUnlock()

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

func (as *AISystem) processFriendlyNPC(w *ecs.World, npcEntity ecs.Entity, npc *components.NPC) {
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
	// Guards could intervene in fights or warn troublemakers
	// For now, just act friendly
	as.processFriendlyNPC(w, npcEntity, npc)
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
