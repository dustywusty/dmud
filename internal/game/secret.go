package game

import (
	"dmud/internal/common"
	"dmud/internal/components"

	"github.com/rs/zerolog/log"
)

const secretAreaID = "99"

func handleXyzzy(player *components.Player, args []string, game *Game) {
	targetComponent, err := game.world.GetComponent(common.EntityID(secretAreaID), "Area")
	if err != nil {
		log.Error().Err(err).Msg("Secret area missing")
		player.Broadcast("Nothing happens.")
		return
	}

	targetArea, ok := targetComponent.(*components.Area)
	if !ok {
		log.Error().Msg("Secret area component type mismatch")
		player.Broadcast("Nothing happens.")
		return
	}

	if player.Area == targetArea {
		player.Broadcast("You feel a brief shimmer, but nothing changes.")
		return
	}

	if player.Area != nil {
		player.Area.RemovePlayer(player)
	}

	player.Area = targetArea
	targetArea.AddPlayer(player)

	player.Broadcast("Reality folds around you, revealing a hidden sanctuary between moments.")
	player.Look(game.world.AsWorldLike())
}
