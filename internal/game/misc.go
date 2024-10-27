package game

import (
	"dmud/internal/components"
	"fmt"

	"github.com/jedib0t/go-pretty/table"
	"github.com/rs/zerolog/log"
)

func handleLook(player *components.Player, args []string, game *Game) {
	player.Look()
}

func handleWho(player *components.Player, args []string, game *Game) {
	game.playersMu.Lock()
	defer game.playersMu.Unlock()

	tw := table.NewWriter()
	tw.SetStyle(table.StyleLight)
	tw.AppendHeader(table.Row{"Player", "Race", "Class", "Online Since"})

	for _, playerEntity := range game.players {
		playerComponent, err := game.world.GetComponent(playerEntity.ID, "Player")
		if err != nil {
			log.Error().Err(err).Msgf("Could not get component for player %s", playerEntity.ID)
			continue
		}
		playerData, ok := playerComponent.(*components.Player)
		if !ok {
			log.Error().Msgf("Error type asserting component for player %s", playerEntity.ID)
			continue
		}
		tw.AppendRow(table.Row{playerData.Name, "??", "??", playerEntity.CreatedAt.DiffForHumans()})
	}

	player.Broadcast(tw.Render())
}

func handleExit(player *components.Player, args []string, game *Game) {
	player.RWMutex.RLock()
	defer player.RWMutex.RUnlock()

	game.HandleDisconnect(player.Client)
}

func handleName(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Usage: name <new_name>")
		return
	}
	newName := args[0]
	game.HandleRename(player, newName)
}

func (g *Game) HandleRename(player *components.Player, newName string) {
	player.Lock()
	defer player.Unlock()

	if newName == "" {
		player.Broadcast(player.Name)
		return
	}

	oldName := player.Name

	g.playersMu.Lock()
	defer g.playersMu.Unlock()

	if _, exists := g.players[newName]; exists {
		player.Broadcast(fmt.Sprintf("The name %s is already taken.", newName))
		return
	}

	player.Name = newName
	g.players[newName] = g.players[oldName]
	delete(g.players, oldName)

	g.Broadcast(fmt.Sprintf("%s has changed their name to %s", oldName, player.Name))
}
