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

	area, err := ecs.GetTypedComponent[*components.Area](w, spawnEntity.ID, "Area")
	if err != nil {
		log.Error().Err(err).Msg("Error getting area component")
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
				ss.spawnNPC(w, area, config, spawn)
			}
		}
	}
}

func (ss *SpawnSystem) countActiveNPCs(w *ecs.World, spawn *components.Spawn, templateID string) int {
	entityIDs, exists := spawn.ActiveSpawns[templateID]
	if !exists {
		return 0
	}

	// Clean up dead references and count valid ones
	validIDs := make([]common.EntityID, 0, len(entityIDs))
	for _, entityID := range entityIDs {
		if _, err := w.FindEntity(entityID); err == nil {
			validIDs = append(validIDs, entityID)
		}
	}

	// Update the list with only valid IDs
	spawn.ActiveSpawns[templateID] = validIDs
	return len(validIDs)
}

func (ss *SpawnSystem) spawnNPC(w *ecs.World, area *components.Area, config components.SpawnConfig, spawn *components.Spawn) {
	template, exists := components.NPCTemplates[config.TemplateID]
	if !exists {
		log.Error().Msgf("NPC template not found: %s", config.TemplateID)
		return
	}

	// Count existing NPCs of this type in the area
	npcEntities, _ := w.FindEntitiesByComponentPredicate("NPC", func(i interface{}) bool {
		n, ok := i.(*components.NPC)
		return ok && n.Area == area && n.TemplateID == template.ID
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
		Name:         template.Name,
		Description:  template.Description,
		Area:         area,
		TemplateID:   template.ID,
		Behavior:     template.Behavior,
		Dialogue:     template.Dialogue,
		LastAction:   time.Now(),
		LastMovement: time.Now(),
	}
	w.AddComponent(&npcEntity, npc)

	// Add Health component
	health := &components.Health{
		Current: template.Health,
		Max:     template.Health,
		Status:  components.Healthy,
	}
	w.AddComponent(&npcEntity, health)

	// Add Combat component for NPCs that can fight on their own
	if template.Behavior == components.BehaviorAggressive || template.Behavior == components.BehaviorGuard {
		combat := &components.Combat{
			MinDamage: template.MinDamage,
			MaxDamage: template.MaxDamage,
		}
		w.AddComponent(&npcEntity, combat)
	}

	// Track the spawn - append to the list
	if spawn.ActiveSpawns[template.ID] == nil {
		spawn.ActiveSpawns[template.ID] = make([]common.EntityID, 0)
	}
	spawn.ActiveSpawns[template.ID] = append(spawn.ActiveSpawns[template.ID], npcEntity.ID)

	// Announce spawn to area (with newline to avoid interrupting typing)
	area.Broadcast(template.Name + " arrives.")

	log.Info().Msgf("Spawned NPC: %s in area %s", template.Name, spawn.AreaID)
}
