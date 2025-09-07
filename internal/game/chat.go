package game

import (
	"dmud/internal/components"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

func handleSay(player *components.Player, args []string, game *Game) {
	msg := strings.Join(args, " ")
	if msg == "" {
		player.Broadcast("Say what?")
		return
	}
	player.Broadcast("You say: " + msg) // echo to speaker
	player.Room.Broadcast(fmt.Sprintf("%s says: %s", player.Name, msg), player)
}

func handleShout(player *components.Player, args []string, game *Game) {
	msg := strings.Join(args, " ")
	if msg == "" {
		player.Broadcast("Shout what?")
		return
	}
	game.HandleShout(player, msg)
}

func (g *Game) HandleShout(player *components.Player, msg string) {
	player.RWMutex.RLock()
	defer player.RWMutex.RUnlock()

	if player.Room == nil {
		player.Broadcast("You shout but there is no sound.")
		return
	}
	log.Info().Msgf("Shout: %s", msg)

	depth := 10

	visited := make(map[*components.Room]bool)
	queue := []*components.Room{player.Room}

	for depth > 0 && len(queue) > 0 {
		depth--
		nextQueue := []*components.Room{}

		for _, room := range queue {
			visited[room] = true
			for _, exit := range room.Exits {
				if !visited[exit.Room] {
					visited[exit.Room] = true
					nextQueue = append(nextQueue, exit.Room)
				}
			}
		}
		queue = nextQueue
	}

	for room := range visited {
		room.Broadcast(fmt.Sprintf("%s shouts: %s", player.Name, msg), player)
	}
}
