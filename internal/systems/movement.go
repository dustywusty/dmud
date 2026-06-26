package systems

import (
	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

type MovementSystem struct{}

func (ms *MovementSystem) Update(w *ecs.World, deltaTime float64) {
	movingEntities, err := w.FindEntitiesByComponentPredicate("Movement", func(i interface{}) bool {
		return true
	})
	if err != nil {
		return
	}

	for _, movingEntity := range movingEntities {
		HandleMovement(w, movingEntity)
	}
}

func HandleMovement(w *ecs.World, movingEntity ecs.Entity) {
	defer func() {
		w.RemoveComponent(movingEntity.ID, "Movement")
	}()

	movingPlayer, err := ecs.GetTypedComponent[*components.Player](w, movingEntity.ID, "Player")
	if err != nil {
		log.Error().Msgf("Error getting moving player component: %v", err)
		return
	}

	playerHealth, err := ecs.GetTypedComponent[*components.Health](w, movingEntity.ID, "Health")
	if err != nil {
		log.Error().Msgf("Error getting player health component: %v", err)
		return
	}

	if playerHealth.Status == components.Dead {
		movingPlayer.Broadcast("You are dead.")
		return
	}

	moving, err := ecs.GetTypedComponent[*components.Movement](w, movingEntity.ID, "Movement")
	if err != nil {
		log.Error().Msgf("Error getting moving component: %v", err)
		return
	}

	if moving.Status == components.Standing {
		return
	}

	area := movingPlayer.Area
	if area == nil {
		log.Warn().Msgf("%v moving, but not in an area", movingPlayer)
		movingPlayer.Broadcast("Do you know where you are?")
		return
	}

	exit := area.GetExit(moving.Direction)
	if exit == nil {
		movingPlayer.Broadcast("You can't go that way.")
		return
	}

	// Size gate: a creature too large can't squeeze through a narrow exit.
	if exit.MaxSize > 0 {
		size := components.SizeMedium
		if stats, err := ecs.GetTypedComponent[*components.Stats](w, movingEntity.ID, "Stats"); err == nil && stats != nil {
			size = stats.Size
		}
		if size > exit.MaxSize {
			movingPlayer.Broadcast(fmt.Sprintf("You're too %s to squeeze through there.", size))
			return
		}
	}

	area.RemovePlayer(movingPlayer)

	movingPlayer.Area = exit.Area
	movingPlayer.Area.AddPlayer(movingPlayer)

	// Charmed/controlled minions tag along into the new room.
	MoveControlledFollowers(w, movingPlayer, movingEntity.ID, area, exit.Area)

	// Roaming trains Dexterity.
	if stats, err := ecs.GetTypedComponent[*components.Stats](w, movingEntity.ID, "Stats"); err == nil {
		components.TrainStat(movingPlayer, stats, components.DEX)
	}

	movingPlayer.Look(w.AsWorldLike())

	log.Debug().Msgf("Moving player %s to %s. Broadcasting state.", movingPlayer.Name, movingPlayer.Area.Description)
	movingPlayer.BroadcastState(w.AsWorldLike(), movingEntity.ID)
}

// MoveControlledFollowers drags any NPC the moving player has charmed or
// controlled from the old room into the new one, so minions follow their master.
// It is exported so other movement paths (e.g. mounted travel) can reuse it.
func MoveControlledFollowers(w *ecs.World, master *components.Player, masterID common.EntityID, from, to *components.Area) {
	if from == nil || to == nil || from == to {
		return
	}

	var followed []string
	for _, npc := range from.GetNPCs(w.AsWorldLike()) {
		entities, _ := w.FindEntitiesByComponentPredicate("NPC", func(i interface{}) bool {
			n, ok := i.(*components.NPC)
			return ok && n == npc
		})
		if len(entities) == 0 {
			continue
		}
		se, err := ecs.GetTypedComponent[*components.StatusEffects](w, entities[0].ID, "StatusEffects")
		if err != nil || se == nil || !se.ControlledBy(masterID) {
			continue
		}

		npc.Area = to
		from.Broadcast(fmt.Sprintf("%s leaves, following %s.", npc.Name, master.Name))
		to.Broadcast(fmt.Sprintf("%s arrives, following %s.", npc.Name, master.Name), master)
		followed = append(followed, npc.Name)
	}

	if len(followed) > 0 {
		master.Broadcast(fmt.Sprintf("Your thralls follow you: %s.", strings.Join(followed, ", ")))
		from.MarkDirty()
		to.MarkDirty()
	}
}
