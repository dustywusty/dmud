package systems

import (
	"dmud/internal/components"
	"dmud/internal/ecs"
	"fmt"
)

type StatusEffectSystem struct{}

func NewStatusEffectSystem() *StatusEffectSystem {
	return &StatusEffectSystem{}
}

func (ses *StatusEffectSystem) Update(w *ecs.World, deltaTime float64) {
	entities, err := w.FindEntitiesByComponentPredicate("StatusEffects", func(i interface{}) bool {
		return true
	})
	if err != nil {
		return
	}

	for _, entity := range entities {
		statusEffects, err := ecs.GetTypedComponent[*components.StatusEffects](w, entity.ID, "StatusEffects")
		if err != nil {
			continue
		}

		removed := statusEffects.RemoveExpired()

		if len(removed) == 0 {
			continue
		}

		player, err := ecs.GetTypedComponent[*components.Player](w, entity.ID, "Player")
		if err != nil {
			continue
		}

		health, err := ecs.GetTypedComponent[*components.Health](w, entity.ID, "Health")
		if err != nil {
			continue
		}

		for _, effect := range removed {
			if effect.HPBonus > 0 {
				health.Lock()
				health.Current -= effect.HPBonus
				if health.Current < 1 {
					health.Current = 1
				}
				health.Unlock()

				player.Broadcast(fmt.Sprintf("The %s has worn off. (-%d HP)", effect.Name, effect.HPBonus))
			}
		}
	}
}
