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
	// Get player entity and quest component
	playerEntity, err := g.getPlayerEntity(player)
	if err != nil {
		log.Error().Err(err).Msg("Error getting player entity")
		return
	}

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

	// Check quest status
	questID := "goblin_ears"
	quest := components.QuestRegistry[questID]
	status := quests.GetQuestStatus(questID)

	switch status {
	case components.QuestStatusNotStarted:
		// Initial greeting with keyword hint
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

	// Find NPCs in the area
	npcs := player.Area.GetNPCs(g.world.AsWorldLike())

	// Check if merchant is present for quest-related keywords
	var merchant *components.NPC
	for _, npc := range npcs {
		if npc.TemplateID == "merchant" {
			merchant = npc
			break
		}
	}

	if merchant == nil {
		return // No merchant to respond
	}

	// Get player entity and quest component
	playerEntity, err := g.getPlayerEntity(player)
	if err != nil {
		log.Error().Err(err).Msg("Error getting player entity")
		return
	}

	questsComp, err := g.world.GetComponent(playerEntity, "PlayerQuests")
	var quests *components.PlayerQuests
	if err != nil {
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

	questID := "goblin_ears"
	quest := components.QuestRegistry[questID]
	status := quests.GetQuestStatus(questID)

	// Handle different keywords
	switch keyword {
	case "work":
		if status == components.QuestStatusNotStarted {
			player.Broadcast(fmt.Sprintf("%s says: Yes! The [goblin] menace has been growing lately.", merchant.Name))
			player.Broadcast(fmt.Sprintf("%s says: They've been raiding caravans and stealing supplies.", merchant.Name))
		}

	case "goblin", "goblins":
		if status == components.QuestStatusNotStarted {
			player.Broadcast(fmt.Sprintf("%s says: Those vile creatures have become quite bold.", merchant.Name))
			player.Broadcast(fmt.Sprintf("%s says: I need proof that they're being dealt with. Bring me 10 [goblin ears].", merchant.Name))
		} else if status == components.QuestStatusInProgress {
			player.Broadcast(fmt.Sprintf("%s says: You'll find them to the south. Bring me 10 of their ears.", merchant.Name))
		}

	case "goblin ears", "goblin ear", "ears":
		if status == components.QuestStatusNotStarted {
			player.Broadcast(fmt.Sprintf("%s says: Bring me 10 goblin ears and I'll give you a proper [reward].", merchant.Name))
		} else if status == components.QuestStatusInProgress {
			playerInvComp, err := g.world.GetComponent(playerEntity, "Inventory")
			if err != nil {
				return
			}
			inventory := playerInvComp.(*components.Inventory)
			item := inventory.FindItem("goblin_ear")
			if item != nil {
				player.Broadcast(fmt.Sprintf("%s says: You have %d goblin ears so far. Keep hunting!", merchant.Name, item.Quantity))
			} else {
				player.Broadcast(fmt.Sprintf("%s says: You haven't collected any yet. The goblins are to the south.", merchant.Name))
			}
		}

	case "reward":
		if status == components.QuestStatusNotStarted {
			player.Broadcast(fmt.Sprintf("%s says: I'll give you a full set of leather armor and 50 gold pieces.", merchant.Name))
			player.Broadcast(fmt.Sprintf("%s says: Do we have a [deal]?", merchant.Name))
		} else if status == components.QuestStatusInProgress {
			// Check if player has the items
			playerInvComp, err := g.world.GetComponent(playerEntity, "Inventory")
			if err != nil {
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
				// Complete the quest
				g.completeGoblinQuest(player, merchant, quest, quests, inventory)
			} else {
				player.Broadcast(fmt.Sprintf("%s says: You don't have enough goblin ears yet. Come back when you have 10.", merchant.Name))
			}
		}

	case "deal":
		if status == components.QuestStatusNotStarted {
			quests.AddQuest(questID)
			player.Broadcast(fmt.Sprintf("Quest accepted: %s", quest.Name))
			player.Broadcast(fmt.Sprintf("%s says: Excellent! I'll be waiting here. Good hunting!", merchant.Name))
		} else {
			player.Broadcast(fmt.Sprintf("%s says: We already have a deal, friend!", merchant.Name))
		}
	}
}

func (g *Game) completeGoblinQuest(player *components.Player, merchant *components.NPC, quest components.Quest, quests *components.PlayerQuests, inventory *components.Inventory) {
	// Verify player has all required items
	for _, req := range quest.Requirements {
		item := inventory.FindItem(req.ItemID)
		if item == nil || item.Quantity < req.Quantity {
			player.Broadcast(fmt.Sprintf("You don't have enough %s! (need %d)",
				components.ItemTemplates[req.ItemID].Name, req.Quantity))
			return
		}
	}

	// Remove required items
	for _, req := range quest.Requirements {
		inventory.RemoveItem(req.ItemID, req.Quantity)
	}

	// Give rewards
	rewardNames := make([]string, 0)
	for _, reward := range quest.Rewards {
		item := components.CreateItem(reward.ItemID, reward.Quantity)
		if item != nil {
			if inventory.AddItem(item) {
				if item.Stackable && item.Quantity > 1 {
					rewardNames = append(rewardNames, fmt.Sprintf("%s x%d", item.Name, item.Quantity))
				} else {
					rewardNames = append(rewardNames, item.Name)
				}
			}
		}
	}

	// Mark quest as completed
	quests.CompleteQuest(quest.ID)

	player.Broadcast(fmt.Sprintf("%s says: Excellent work! Here is your reward.", merchant.Name))
	player.Broadcast(fmt.Sprintf("Quest completed: %s", quest.Name))
	player.Broadcast(fmt.Sprintf("You received: %s", strings.Join(rewardNames, ", ")))
	player.Area.Broadcast(fmt.Sprintf("%s has completed a quest!", player.Name), player)
}
