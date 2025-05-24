package game

import (
	"dmud/internal/components"
	"dmud/internal/util"
	"strings"

	"github.com/rs/zerolog/log"
)

// handleKill processes the kill command.
func handleKill(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Kill whom?")
		return
	}
	targetName := strings.Join(args, " ")
	game.HandleKill(player, targetName)
}

// HandleKill processes the kill action for a player.
func (g *Game) HandleKill(player *components.Player, targetName string) {
    log.Trace().Msgf("Kill: %s", targetName)

    // First check for players
    g.playersMu.Lock()
    targetEntity := g.players[targetName]
    playerEntity := g.players[player.Name]
    g.playersMu.Unlock()

    if targetEntity == nil {
        // Check for NPCs
        npcs := player.Room.GetNPCs(g.world)
        for _, npc := range npcs {
            if strings.Contains(strings.ToLower(npc.Name), strings.ToLower(targetName)) {
                // Find NPC entity
                npcEntities, _ := g.world.FindEntitiesByComponentPredicate("NPC", func(i interface{}) bool {
                    n, ok := i.(*components.NPC)
                    return ok && n == npc
                })

                if len(npcEntities) > 0 {
                    targetEntity = &npcEntities[0]
                    break
                }
            }
        }

        if targetEntity == nil {
            player.Broadcast("They aren't here.")
            return
        }
    }

    if playerEntity == nil {
        log.Warn().Msgf("Error getting player's own entity for %s", player.Name)
        return
    }

    combatComponent := &components.Combat{
        TargetID:  targetEntity.ID,
        MinDamage: 10,
        MaxDamage: 50,
    }

    g.world.AddComponent(playerEntity, combatComponent)

    // Announce combat
    if npc, err := util.GetTypedComponent[*components.NPC](g.world, targetEntity.ID, "NPC"); err == nil {
        player.Room.Broadcast(player.Name + " attacks " + npc.Name + "!")
    }
}
