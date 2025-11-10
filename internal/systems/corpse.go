package systems

import (
	"dmud/internal/components"
	"dmud/internal/ecs"

	"github.com/rs/zerolog/log"
)

// CorpseSystem handles corpse decay and cleanup
type CorpseSystem struct{}

func NewCorpseSystem() *CorpseSystem {
	return &CorpseSystem{}
}

func (cs *CorpseSystem) Update(w *ecs.World, deltaTime float64) {
	// Find all corpse entities
	corpseEntities, err := w.FindEntitiesByComponentPredicate("Corpse", func(i interface{}) bool {
		return true
	})
	if err != nil || len(corpseEntities) == 0 {
		return
	}

	for _, entity := range corpseEntities {
		corpse, err := ecs.GetTypedComponent[*components.Corpse](w, entity.ID, "Corpse")
		if err != nil {
			continue
		}

		// Check if corpse has decayed
		if corpse.IsDecayed() {
			corpse.RLock()
			area := corpse.Area
			victimName := corpse.VictimName
			corpse.RUnlock()

			// Broadcast decay message to area
			if area != nil {
				area.Broadcast("The corpse of " + victimName + " crumbles to dust.")
			}

			// Remove corpse entity from world
			w.RemoveEntity(entity.ID)

			log.Debug().Msgf("Corpse of %s has decayed and been removed", victimName)
		}
	}
}
