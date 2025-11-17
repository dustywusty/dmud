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
	player.Area.Broadcast(fmt.Sprintf("%s says: %s", player.Name, msg), player)

	// Check if this triggers NPC keyword responses
	game.handleSayToNPC(player, msg)
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

	if player.Area == nil {
		player.Broadcast("You shout but there is no sound.")
		return
	}
	log.Info().Msgf("Shout: %s", msg)

	depth := 10

	visited := make(map[*components.Area]bool)
	queue := []*components.Area{player.Area}

	for depth > 0 && len(queue) > 0 {
		depth--
		nextQueue := []*components.Area{}

		for _, area := range queue {
			visited[area] = true
			for _, exit := range area.Exits {
				if !visited[exit.Area] {
					visited[exit.Area] = true
					nextQueue = append(nextQueue, exit.Area)
				}
			}
		}
		queue = nextQueue
	}

	for area := range visited {
		area.Broadcast(fmt.Sprintf("%s shouts: %s", player.Name, msg), player)
	}
}
