package systems

import (
	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/util"
	"fmt"
	"strings"
	"time"
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

	now := time.Now()
	for _, entity := range entities {
		statusEffects, err := ecs.GetTypedComponent[*components.StatusEffects](w, entity.ID, "StatusEffects")
		if err != nil {
			continue
		}

		player, _ := ecs.GetTypedComponent[*components.Player](w, entity.ID, "Player")
		npc, _ := ecs.GetTypedComponent[*components.NPC](w, entity.ID, "NPC")
		health, _ := ecs.GetTypedComponent[*components.Health](w, entity.ID, "Health")

		// 1) Periodic damage/heal (burning, poison, regeneration).
		stateDirty := false
		if ticks := statusEffects.Tick(now); len(ticks) > 0 && health != nil {
			maxHP := health.Max + statusEffects.GetTotalHPBonus()
			var killer common.EntityID
			for _, t := range ticks {
				health.Current += t.HPDelta
				if health.Current > maxHP {
					health.Current = maxHP
				}
				if t.HPDelta < 0 {
					killer = t.Source
					if player != nil {
						player.Broadcast(util.TagMessage("DMG", fmt.Sprintf("%s sears you for %d damage!", t.Name, -t.HPDelta)))
					} else if npc != nil && npc.Area != nil {
						npc.Area.Broadcast(fmt.Sprintf("%s writhes as %s takes hold.", npc.Name, strings.ToLower(t.Name)))
					}
				} else if t.HPDelta > 0 && player != nil {
					player.Broadcast(util.TagMessage("STATUS", fmt.Sprintf("%s knits your wounds for %d.", t.Name, t.HPDelta)))
				}
			}
			stateDirty = true
			if health.Current <= 0 {
				health.Current = 0
				applyEffectDeath(w, killer, entity.ID, player, npc)
				continue // entity is dead/revived; nothing more to do this pass
			}
		}

		// 2) Expiry of timed effects.
		removed := statusEffects.RemoveExpired()
		if len(removed) > 0 {
			stateDirty = true
		}

		for _, effect := range removed {
			if effect.HPBonus > 0 && health != nil {
				health.Current -= effect.HPBonus
				if health.Current < 1 {
					health.Current = 1
				}
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
				npcArea := npc.Area
				npcName := npc.Name
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

		if player != nil && stateDirty {
			// Push fresh vitals when DoT/regen ticked or an effect expired.
			player.BroadcastState(w.AsWorldLike(), entity.ID)
		}
	}
}

// applyEffectDeath routes a kill from a damage-over-time effect through the normal
// combat death path (corpse, XP for the source if any, respawn). A zero source
// (e.g. environmental) yields an unattributed death.
func applyEffectDeath(w *ecs.World, attackerID, targetID common.EntityID, targetPlayer *components.Player, targetNPC *components.NPC) {
	var attackerPlayer *components.Player
	var attackerNPC *components.NPC
	if attackerID != "" {
		attackerPlayer, _ = ecs.GetTypedComponent[*components.Player](w, attackerID, "Player")
		attackerNPC, _ = ecs.GetTypedComponent[*components.NPC](w, attackerID, "NPC")
	}
	handleTargetDeath(w.AsWorldLike(), attackerID, targetID, attackerPlayer, targetPlayer, attackerNPC, targetNPC)
}
