// internal/systems/spawn.go
package systems

import (
	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
	"math/rand"
	"time"

	"github.com/rs/zerolog/log"
)

type SpawnSystem struct {
	lastCheck time.Time
}

func NewSpawnSystem() *SpawnSystem {
	return &SpawnSystem{
		lastCheck: time.Now(),
	}
}

func (ss *SpawnSystem) Update(w *ecs.World, deltaTime float64) {
	// Only check spawns every 5 seconds
	if time.Since(ss.lastCheck) < 5*time.Second {
		return
	}
	ss.lastCheck = time.Now()

	// Find all entities with Spawn components
	spawnEntities, err := w.FindEntitiesByComponentPredicate("Spawn", func(i interface{}) bool {
		return true
	})
	if err != nil {
		log.Error().Err(err).Msg("Error finding spawn entities")
		return
	}

	for _, entity := range spawnEntities {
		ss.processSpawn(w, entity)
	}
}

func (ss *SpawnSystem) processSpawn(w *ecs.World, spawnEntity ecs.Entity) {
	spawn, err := ecs.GetTypedComponent[*components.Spawn](w, spawnEntity.ID, "Spawn")
	if err != nil {
		log.Error().Err(err).Msg("Error getting spawn component")
		return
	}

	room, err := ecs.GetTypedComponent[*components.Room](w, spawnEntity.ID, "Room")
	if err != nil {
		log.Error().Err(err).Msg("Error getting room component")
		return
	}

	spawn.Lock()
	defer spawn.Unlock()

	for _, config := range spawn.Configs {
		if config.Type != components.SpawnTypeNPC {
			continue // For now, only handle NPCs
		}

		// Count active spawns of this type
		activeCount := ss.countActiveNPCs(w, spawn, config.TemplateID)

		// Check if we need to spawn more
		if activeCount < config.MinCount {
			// Check spawn chance
			if rand.Float64() <= config.Chance {
				ss.spawnNPC(w, room, config, spawn)
			}
		}
	}
}

func (ss *SpawnSystem) countActiveNPCs(w *ecs.World, spawn *components.Spawn, templateID string) int {
	count := 0
	for tid, entityID := range spawn.ActiveSpawns {
		if tid != templateID {
			continue
		}

		// Check if entity still exists
		if _, err := w.FindEntity(entityID); err == nil {
			count++
		} else {
			// Clean up dead reference
			delete(spawn.ActiveSpawns, tid)
		}
	}
	return count
}

func (ss *SpawnSystem) spawnNPC(w *ecs.World, room *components.Room, config components.SpawnConfig, spawn *components.Spawn) {
	template, exists := components.NPCTemplates[config.TemplateID]
	if !exists {
		log.Error().Msgf("NPC template not found: %s", config.TemplateID)
		return
	}

	// Check if we've already spawned this NPC type
	if existingID, exists := spawn.ActiveSpawns[template.ID]; exists {
		// Verify the entity still exists
		if _, err := w.GetComponent(common.EntityID(existingID), "NPC"); err == nil {
			// NPC still exists, don't spawn another
			return
		}
		// Entity no longer exists, remove from tracking
		delete(spawn.ActiveSpawns, template.ID)
	}

	// Count existing NPCs of this type in the room
	npcEntities, _ := w.FindEntitiesByComponentPredicate("NPC", func(i interface{}) bool {
		n, ok := i.(*components.NPC)
		return ok && n.Room == room && n.TemplateID == template.ID
	})

	// Only spawn if under the max count
	if len(npcEntities) >= config.MaxCount {
		return
	}

	// Create NPC entity
	npcEntity := ecs.NewEntity()
	w.AddEntity(npcEntity)

	// Add NPC component
	npc := &components.NPC{
		Name:        template.Name,
		Description: template.Description,
		Room:        room,
		TemplateID:  template.ID,
		Behavior:    template.Behavior,
		Dialogue:    template.Dialogue,
		LastAction:  time.Now(),
	}
	w.AddComponent(&npcEntity, npc)

	// Add Health component
	health := &components.Health{
		Current: template.Health,
		Max:     template.Health,
		Status:  components.Healthy,
	}
	w.AddComponent(&npcEntity, health)

	// Add Combat component for aggressive NPCs
	if template.Behavior == components.BehaviorAggressive {
		combat := &components.Combat{
			MinDamage: template.MinDamage,
			MaxDamage: template.MaxDamage,
		}
		w.AddComponent(&npcEntity, combat)
	}

	// Track the spawn
	spawn.ActiveSpawns[template.ID] = npcEntity.ID

	// Announce spawn to room (with newline to avoid interrupting typing)
	room.Broadcast("\n" + template.Name + " arrives.")

	log.Info().Msgf("Spawned NPC: %s in room %s", template.Name, spawn.RoomID)
}
