package game

import (
	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

func handleKill(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Kill whom?")
		return
	}
	targetName := strings.Join(args, " ")

	// Check for "kill all" command
	if strings.ToLower(targetName) == "all" {
		game.HandleKillAll(player)
		return
	}

	game.HandleKill(player, targetName)
}

func (g *Game) HandleKillAll(player *components.Player) {
	log.Trace().Msgf("Kill all by %s", player.Name)

	g.playersMu.Lock()
	playerEntity := g.players[player.Name]
	g.playersMu.Unlock()

	if playerEntity == nil {
		log.Warn().Msgf("Error getting player's own entity for %s", player.Name)
		return
	}

	// Get all NPCs in the area
	npcs := player.Area.GetNPCs(g.world.AsWorldLike())
	if len(npcs) == 0 {
		player.Broadcast("There's nothing here to attack.")
		return
	}

	// Find all NPC entities in the area
	var targetEntityIDs []common.EntityID
	for _, npc := range npcs {
		npcEntities, _ := g.world.FindEntitiesByComponentPredicate("NPC", func(i interface{}) bool {
			n, ok := i.(*components.NPC)
			return ok && n == npc
		})
		if len(npcEntities) > 0 {
			targetEntityIDs = append(targetEntityIDs, npcEntities[0].ID)
		}
	}

	if len(targetEntityIDs) == 0 {
		player.Broadcast("There's nothing here to attack.")
		return
	}

	// Set player to attack the first target, queue the rest
	combatComponent := &components.Combat{
		TargetID:    targetEntityIDs[0],
		TargetQueue: targetEntityIDs[1:], // Queue up the rest
		MinDamage:   10,
		MaxDamage:   50,
	}
	g.world.AddComponent(playerEntity, combatComponent)

	// Announce combat
	player.Area.Broadcast(player.Name + " attacks everything in sight!")
	if len(targetEntityIDs) == 1 {
		player.Broadcast("You engage 1 enemy!")
	} else {
		player.Broadcast(fmt.Sprintf("You engage %d enemies!", len(targetEntityIDs)))
	}
}

func (g *Game) HandleKill(player *components.Player, targetName string) {
	log.Trace().Msgf("Kill: %s", targetName)

	// First check for players
	g.playersMu.Lock()
	defer g.playersMu.Unlock()

	targetEntity := g.players[targetName]
	playerEntity := g.players[player.Name]

	if targetEntity == nil {
		// Check for NPCs
		npcs := player.Area.GetNPCs(g.world.AsWorldLike())
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
	if npc, err := ecs.GetTypedComponent[*components.NPC](g.world, targetEntity.ID, "NPC"); err == nil {
		player.Area.Broadcast(player.Name + " attacks " + npc.Name + "!")
	}
}
