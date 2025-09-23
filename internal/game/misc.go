package game

import (
	"dmud/internal/components"
	"fmt"
	"strings"

	"github.com/jedib0t/go-pretty/table"
	"github.com/rs/zerolog/log"
)

func handleLook(player *components.Player, args []string, game *Game) {
	player.Look(game.world.AsWorldLike())
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
	newName = strings.TrimSpace(newName)
	if newName == "" {
		player.Broadcast("Usage: name <new_name>")
		return
	}

	player.Lock()
	oldName := player.Name
	player.Unlock()

	g.playersMu.Lock()
	if _, exists := g.players[newName]; exists {
		g.playersMu.Unlock()
		player.Broadcast(fmt.Sprintf("The name %s is already taken.", newName))
		return
	}
	ent := g.players[oldName]
	delete(g.players, oldName)
	g.players[newName] = ent
	g.playersMu.Unlock()

	player.Lock()
	player.Name = newName
	player.Unlock()

	g.Broadcast(fmt.Sprintf("%s has changed their name to %s", oldName, newName))
}

// internal/game/misc.go - Add examine command

func handleExamine(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Examine what?")
		return
	}

	target := strings.Join(args, " ")

	// Check for NPCs in the area
	npcs := player.Area.GetNPCs(game.world.AsWorldLike())
	for _, npc := range npcs {
		if strings.Contains(strings.ToLower(npc.Name), strings.ToLower(target)) {
			player.Broadcast(npc.Description)

			// Show health status
			npcEntities, _ := game.world.FindEntitiesByComponentPredicate("NPC", func(i interface{}) bool {
				n, ok := i.(*components.NPC)
				return ok && n == npc
			})

			if len(npcEntities) > 0 {
				health, err := game.world.GetComponent(npcEntities[0].ID, "Health")
				if err == nil {
					h := health.(*components.Health)
					healthPercent := float64(h.Current) / float64(h.Max) * 100

					var status string
					switch {
					case healthPercent >= 90:
						status = "is in excellent condition"
					case healthPercent >= 70:
						status = "has a few scratches"
					case healthPercent >= 50:
						status = "is wounded"
					case healthPercent >= 30:
						status = "is badly wounded"
					case healthPercent >= 10:
						status = "is near death"
					default:
						status = "is dying"
					}

					player.Broadcast(npc.Name + " " + status + ".")
				}
			}

			return
		}
	}

	// Check for players
	targetPlayer := player.Area.GetPlayer(target)
	if targetPlayer != nil {
		player.Broadcast("You see " + targetPlayer.Name + ", a fellow adventurer.")
		return
	}

	player.Broadcast("You don't see that here.")
}
