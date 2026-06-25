package game

import (
	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/util"
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
	minD, maxD := g.playerMeleeDamage(player)
	combatComponent := &components.Combat{
		TargetID:    targetEntityIDs[0],
		TargetQueue: targetEntityIDs[1:], // Queue up the rest
		MinDamage:   minD,
		MaxDamage:   maxD,
	}
	g.world.AddComponent(playerEntity, combatComponent)

	// Announce combat
	player.Area.Broadcast(util.TagMessage("DMG", player.Name+" attacks everything in sight!"))
	if len(targetEntityIDs) == 1 {
		player.Broadcast(util.TagMessage("DMG", "You engage 1 enemy!"))
	} else {
		player.Broadcast(util.TagMessage("DMG", fmt.Sprintf("You engage %d enemies!", len(targetEntityIDs))))
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
		matcher, _, isPattern, err := buildItemMatcher(targetName)
		if err != nil {
			player.Broadcast(fmt.Sprintf("Invalid target: %v", err))
			return
		}

		var targetEntityIDs []common.EntityID

		// Check for NPCs
		npcs := player.Area.GetNPCs(g.world.AsWorldLike())
		for _, npc := range npcs {
			if matcher(npc.Name) {
				// Find NPC entity
				npcEntities, _ := g.world.FindEntitiesByComponentPredicate("NPC", func(i interface{}) bool {
					n, ok := i.(*components.NPC)
					return ok && n == npc
				})

				if len(npcEntities) > 0 {
					targetEntityIDs = append(targetEntityIDs, npcEntities[0].ID)
					// If it's not a pattern (wildcard/regex), we only want one target
					if !isPattern {
						break
					}
				}
			}
		}

		if len(targetEntityIDs) == 0 {
			player.Broadcast("They aren't here.")
			return
		}

		// Use the first found entity as the primary target
		// We need to fetch the pointer to the entity for the logic below that expects 'targetEntity'
		// Note: This logic for 'targetEntity' variable is a bit mixed now because we have IDs.
		// However, the combat component needs IDs.
		// We can refactor the construction of the Combat component to use the IDs directly.

		minD, maxD := g.playerMeleeDamage(player)
		combatComponent := &components.Combat{
			TargetID:  targetEntityIDs[0],
			MinDamage: minD,
			MaxDamage: maxD,
		}

		if len(targetEntityIDs) > 1 {
			combatComponent.TargetQueue = targetEntityIDs[1:]
			player.Broadcast(util.TagMessage("DMG", fmt.Sprintf("You engage %d enemies!", len(targetEntityIDs))))
		}

		g.world.AddComponent(playerEntity, combatComponent)

		// Announce combat for the prime target
		// We can skip the generic announce code below or modify it.
		// For now, let's just let the rest of the function run if we have a single target,
		// or return early if we've handled the "group" case.

		if len(targetEntityIDs) > 1 {
			// Announce mass attack
			player.Area.Broadcast(util.TagMessage("DMG", player.Name+" attacks multiple enemies!"))
			return
		}

		// Set targetEntity for the single-target fallback logic below
		ent, err := g.world.FindEntity(targetEntityIDs[0])
		if err != nil {
			log.Error().Err(err).Msgf("Failed to find entity %s after initial discovery", targetEntityIDs[0])
			player.Broadcast("Something went wrong targeting that.")
			return
		}
		targetEntity = &ent
	}

	if playerEntity == nil {
		log.Warn().Msgf("Error getting player's own entity for %s", player.Name)
		return
	}

	minD, maxD := g.playerMeleeDamage(player)
	combatComponent := &components.Combat{
		TargetID:  targetEntity.ID,
		MinDamage: minD,
		MaxDamage: maxD,
	}

	g.world.AddComponent(playerEntity, combatComponent)

	// Announce combat
	if npc, err := ecs.GetTypedComponent[*components.NPC](g.world, targetEntity.ID, "NPC"); err == nil {
		player.Area.Broadcast(util.TagMessage("DMG", player.Name+" attacks "+npc.Name+"!"))
	}
}
