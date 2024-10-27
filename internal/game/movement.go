package game

import (
	"dmud/internal/components"

	"github.com/rs/zerolog/log"
)

// createMoveHandler returns a CommandHandler for moving in the given direction.
func (g *Game) createMoveHandler(direction string) CommandHandler {
	return func(player *components.Player, args []string, game *Game) {
		game.HandleMove(player, direction)
	}
}

// HandleMove processes movement commands for a player.
func (g *Game) HandleMove(player *components.Player, direction string) {
	player.RWMutex.RLock()
	playerEntity := g.players[player.Name]
	player.RWMutex.RUnlock()

	if playerEntity == nil {
		log.Warn().Msgf("Error getting player's own entity for %s", player.Name)
		return
	}

	movement := &components.Movement{
		Direction: direction,
		Status:    components.Walking,
	}
	g.world.AddComponent(playerEntity, movement)
}
