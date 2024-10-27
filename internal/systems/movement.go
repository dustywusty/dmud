package systems

import (
	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/util"

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

	movingPlayer, err := util.GetTypedComponent[*components.Player](w, movingEntity.ID, "Player")
	if err != nil {
		log.Error().Msgf("Error getting moving player component: %v", err)
		return
	}

	playerHealth, err := util.GetTypedComponent[*components.Health](w, movingEntity.ID, "Health")
	if err != nil {
		log.Error().Msgf("Error getting player health component: %v", err)
		return
	}

	if playerHealth.Status == components.Dead {
		movingPlayer.Broadcast("You are dead.")
		return
	}

	moving, err := util.GetTypedComponent[*components.Movement](w, movingEntity.ID, "Movement")
	if err != nil {
		log.Error().Msgf("Error getting moving component: %v", err)
		return
	}

	if moving.Status == components.Standing {
		return
	}

	room := movingPlayer.Room
	if room == nil {
		log.Warn().Msgf("%v moving, but not in a room", movingPlayer)
		movingPlayer.Broadcast("Do you know where you are?")
		return
	}

	exit := room.GetExit(moving.Direction)
	if exit == nil {
		movingPlayer.Broadcast("You can't go that way.")
		return
	}

	room.RemovePlayer(movingPlayer)

	movingPlayer.Room = exit.Room
	movingPlayer.Room.AddPlayer(movingPlayer)
	movingPlayer.Broadcast(exit.Room.Description)
}
