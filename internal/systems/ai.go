// internal/systems/ai.go
package systems

import (
	"dmud/internal/components"
	"dmud/internal/ecs"
	"math/rand"
	"time"
)

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

	// Process based on behavior type
	switch npc.Behavior {
	case components.BehaviorAggressive:
		as.processAggressiveNPC(w, npcEntity, npc)
	case components.BehaviorFriendly, components.BehaviorMerchant:
		as.processFriendlyNPC(w, npcEntity, npc)
	case components.BehaviorGuard:
		as.processGuardNPC(w, npcEntity, npc)
	case components.BehaviorPassive:
		as.processPassiveNPC(w, npcEntity, npc)
	}
}

func (as *AISystem) processAggressiveNPC(w *ecs.World, npcEntity ecs.Entity, npc *components.NPC) {
	combat, _ := ecs.GetTypedComponent[*components.Combat](w, npcEntity.ID, "Combat")

	// If not in combat, look for targets
	if combat == nil || combat.TargetID == "" {
		npc.RLock()
		room := npc.Room
		npc.RUnlock()

		if room != nil && len(room.Players) > 0 {
			// Pick a random player to attack
			room.PlayersMutex.RLock()
			if len(room.Players) > 0 {
				target := room.Players[rand.Intn(len(room.Players))]
				room.PlayersMutex.RUnlock()

				// Find player entity
				playerEntities, _ := w.FindEntitiesByComponentPredicate("Player", func(i interface{}) bool {
					p, ok := i.(*components.Player)
					return ok && p == target
				})

				if len(playerEntities) > 0 {
					// Start combat
					newCombat := &components.Combat{
						TargetID:  playerEntities[0].ID,
						MinDamage: combat.MinDamage,
						MaxDamage: combat.MaxDamage,
					}
					w.AddComponent(&npcEntity, newCombat)

					room.Broadcast(npc.Name + " attacks " + target.Name + "!")
				}
			} else {
				room.PlayersMutex.RUnlock()
			}
		}
	}
}

func (as *AISystem) processFriendlyNPC(w *ecs.World, npcEntity ecs.Entity, npc *components.NPC) {
	// Occasionally say something
	if time.Since(npc.LastAction) > 30*time.Second && rand.Float64() < 0.3 {
		dialogue := npc.GetRandomDialogue()
		if dialogue != "" && npc.Room != nil {
			npc.Room.Broadcast(npc.Name + " says: " + dialogue)
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
		if dialogue != "" && npc.Room != nil {
			npc.Room.Broadcast(npc.Name + " " + dialogue)
			npc.Lock()
			defer npc.Unlock()
			npc.LastAction = time.Now()
		}
	}
}
