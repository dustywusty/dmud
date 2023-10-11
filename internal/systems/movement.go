package systems

import (
	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/util"

	"github.com/rs/zerolog/log"
)

type MovementSystem struct{}

func (ms *MovementSystem) Update(w *ecs.World, deltaTime float64) {
	// s := rand.NewSource(time.Now().UnixNano())
	// r := rand.New(s)

	movingEntities, err := w.FindEntitiesByComponentPredicate("Movement", func(i interface{}) bool {
		return true
	})
	if err != nil {
		return
	}

	for _, movingEntity := range movingEntities {
		moving, err := util.GetTypedComponent[components.Movement](w, movingEntity.ID, "Movement")
		if err != nil {
			log.Error().Msgf("Error getting moving component: %v", err)
			return
		}

		if moving.Status == components.Standing {
			continue
		}

		movingPlayer, err := util.GetTypedComponent[components.Player](w, movingEntity.ID, "Player")
		if err != nil {
			log.Error().Msgf("Error getting moving player component: %v", err)
			return
		}

		room := movingPlayer.Room
		if room == nil {
			log.Error().Msgf("Error getting current room for player: %v", movingPlayer)
			return
		}

		exit := room.GetExit(moving.Direction)
		if exit == nil {
			moving.Status = components.Standing
			movingPlayer.Broadcast("You can't go that way.")
			return
		}

		room.RemovePlayer(movingPlayer)

		movingPlayer.Room = exit.Room
		movingPlayer.Room.AddPlayer(movingPlayer)
		movingPlayer.Broadcast(exit.Room.Description)

		moving.Status = components.Standing

		log.Trace().Msgf("Moving player: %v", movingEntity.ID)
	}
}
