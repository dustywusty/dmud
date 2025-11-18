package game

import (
	"dmud/internal/components"
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
)

func (g *Game) handleHail(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Hail whom? Usage: hail <npc>")
		return
	}

	targetName := strings.ToLower(strings.Join(args, " "))

	// Find NPCs in the area
	npcs := player.Area.GetNPCs(g.world.AsWorldLike())
	for _, npc := range npcs {
		if strings.Contains(strings.ToLower(npc.Name), targetName) {
			g.handleNPCHail(player, npc)
			return
		}
	}

	player.Broadcast("You don't see that NPC here.")
}

func (g *Game) handleNPCHail(player *components.Player, npc *components.NPC) {
	// Handle different NPC types
	switch npc.TemplateID {
	case "merchant":
		g.handleMerchantHail(player, npc)
	default:
		// Default hail response
		if len(npc.Dialogue) > 0 {
			player.Broadcast(fmt.Sprintf("%s says: %s", npc.Name, npc.Dialogue[0]))
		} else {
			player.Broadcast(fmt.Sprintf("%s nods at you.", npc.Name))
		}
	}
}

func (g *Game) handleMerchantHail(player *components.Player, npc *components.NPC) {
	// Get player entity
	playerEntity, err := g.getPlayerEntity(player)
	if err != nil {
		log.Error().Err(err).Msg("Error getting player entity")
		return
	}

	// Get or create player quests component
	questsComp, err := g.world.GetComponent(playerEntity, "PlayerQuests")
	var quests *components.PlayerQuests
	if err != nil {
		// Player doesn't have quest component, create it
		quests = components.NewPlayerQuests()
		playerEntities, _ := g.world.FindEntitiesByComponentPredicate("Player", func(i interface{}) bool {
			p, ok := i.(*components.Player)
			return ok && p == player
		})
		if len(playerEntities) > 0 {
			g.world.AddComponent(&playerEntities[0], quests)
		}
	} else {
		quests = questsComp.(*components.PlayerQuests)
	}

	// Get the goblin quest
	questID := "goblin_ears"
	quest := components.QuestRegistry[questID]
	if quest == nil || quest.Dialogue == nil {
		player.Broadcast(fmt.Sprintf("%s nods at you.", npc.Name))
		return
	}

	// Show greeting based on quest status
	status := quests.GetQuestStatus(questID)

	switch status {
	case components.QuestStatusNotStarted:
		player.Broadcast(fmt.Sprintf("%s says: Greetings, adventurer! I have been traveling these lands for many years.", npc.Name))
		player.Broadcast(fmt.Sprintf("%s says: If you are interested in earning some coin, I might have some [work] for you.", npc.Name))

	case components.QuestStatusInProgress:
		// Check if player has required items
		playerInvComp, err := g.world.GetComponent(playerEntity, "Inventory")
		if err != nil {
			player.Broadcast(fmt.Sprintf("%s says: Come back when you have what I need!", npc.Name))
			return
		}

		inventory := playerInvComp.(*components.Inventory)
		hasAllItems := true
		for _, req := range quest.Requirements {
			item := inventory.FindItem(req.ItemID)
			if item == nil || item.Quantity < req.Quantity {
				hasAllItems = false
				break
			}
		}

		if hasAllItems {
			player.Broadcast(fmt.Sprintf("%s says: Ah! You have the goblin ears I requested!", npc.Name))
			player.Broadcast(fmt.Sprintf("%s says: I will gladly give you your [reward] for them.", npc.Name))
		} else {
			player.Broadcast(fmt.Sprintf("%s says: The [goblin ears] I need can be found on the goblins to the south.", npc.Name))
			player.Broadcast(fmt.Sprintf("%s says: Bring me 10 of them and I'll reward you handsomely.", npc.Name))
		}

	case components.QuestStatusCompleted:
		player.Broadcast(fmt.Sprintf("%s says: Thank you again for your help with those goblins!", npc.Name))
	}
}

func (g *Game) handleSayToNPC(player *components.Player, keyword string) {
	keyword = strings.ToLower(strings.TrimSpace(keyword))

	// Get player entity
	playerEntity, err := g.getPlayerEntity(player)
	if err != nil {
		return
	}

	// Use the dialogue handler from components
	handler := &components.QuestDialogueHandler{
		World: g.world.AsWorldLike(),
	}

	handler.HandleKeyword(player, keyword, playerEntity)
}
