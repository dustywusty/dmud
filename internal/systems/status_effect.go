package systems

import (
	"dmud/internal/components"
	"dmud/internal/ecs"
	"fmt"
	"strings"
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

		player, _ := ecs.GetTypedComponent[*components.Player](w, entity.ID, "Player")
		npc, _ := ecs.GetTypedComponent[*components.NPC](w, entity.ID, "NPC")
		health, _ := ecs.GetTypedComponent[*components.Health](w, entity.ID, "Health")

		for _, effect := range removed {
			if effect.HPBonus > 0 && health != nil {
				health.Lock()
				health.Current -= effect.HPBonus
				if health.Current < 1 {
					health.Current = 1
				}
				health.Unlock()
			}

			if player != nil {
				if effect.HPBonus > 0 {
					player.Broadcast(fmt.Sprintf("The %s has worn off. (-%d HP)", effect.Name, effect.HPBonus))
				} else {
					player.Broadcast(fmt.Sprintf("The %s has worn off.", effect.Name))
				}
				continue
			}

			if npc != nil {
				npc.RLock()
				npcArea := npc.Area
				npcName := npc.Name
				npc.RUnlock()
				if npcArea == nil {
					continue
				}

				if effect.Type == components.StatusEffectControlledUndead {
					npcArea.Broadcast(fmt.Sprintf("%s shudders as necromantic control fades.", npcName))
					continue
				}
				if effect.Type == components.StatusEffectCharmed {
					npcArea.Broadcast(fmt.Sprintf("%s blinks and regains free will.", npcName))
					continue
				}

				effectName := strings.TrimSpace(strings.ToLower(effect.Name))
				if effectName == "" {
					effectName = "a lingering effect"
				}
				npcArea.Broadcast(fmt.Sprintf("%s is no longer affected by %s.", npcName, effectName))
			}
		}

		if player != nil {
			// Broadcast state update when player effects are removed.
			player.BroadcastState(w.AsWorldLike(), entity.ID)
		}
	}
}
