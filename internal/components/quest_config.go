package components

import (
	"fmt"

	"dmud/internal/common"

	"github.com/rs/zerolog/log"
)

// InitializeQuests sets up all quest dialogues
func InitializeQuests() {
	goblinQuest := QuestRegistry["goblin_ears"]
	goblinQuest.Dialogue = createGoblinEarsDialogue()
}

// createGoblinEarsDialogue creates the dialogue tree for the goblin ears quest
func createGoblinEarsDialogue() *QuestDialogue {
	return &QuestDialogue{
		NPCName:  "a traveling merchant",
		Greeting: "Welcome, traveler! I've got goods from across the realm. Looking for [work]?",
		Nodes: map[string]*DialogueNode{
			"work": {
				Triggers: []string{"work", "job", "quest"},
				Text:     "Yes! The [goblin] menace has been growing lately. They've been raiding caravans and stealing supplies.",
				Condition: func(player *Player, quest *PlayerQuest) bool {
					return quest.Status == QuestStatusNotStarted
				},
			},
			"goblin": {
				Triggers: []string{"goblin", "goblins", "menace"},
				Text:     "Those vile creatures have become quite bold. I need proof that they're being dealt with. Bring me 10 [goblin ears].",
				Condition: func(player *Player, quest *PlayerQuest) bool {
					return quest.Status == QuestStatusNotStarted
				},
			},
			"goblin ears": {
				Triggers: []string{"goblin ears", "goblin ear", "ears", "proof"},
				Text:     "",
				Condition: func(player *Player, quest *PlayerQuest) bool {
					return true // Always available to check progress
				},
				Action: &QuestAction{
					Type: ActionCheckProgress,
				},
			},
			"deal": {
				Triggers: []string{"deal", "accept", "yes", "agree"},
				Text:     "",
				Condition: func(player *Player, quest *PlayerQuest) bool {
					return quest.Status == QuestStatusNotStarted
				},
				Action: &QuestAction{
					Type: ActionAcceptQuest,
				},
			},
			"reward": {
				Triggers: []string{"reward", "trade", "exchange", "complete"},
				Text:     "",
				Condition: func(player *Player, quest *PlayerQuest) bool {
					return quest.Status == QuestStatusInProgress
				},
				Action: &QuestAction{
					Type: ActionCompleteQuest,
				},
			},
		},
	}
}

// QuestDialogueHandler handles quest dialogue execution with world access
type QuestDialogueHandler struct {
	World WorldLike
}

// HandleKeyword processes a keyword for quest dialogue from any NPC
func (h *QuestDialogueHandler) HandleKeyword(player *Player, keyword string, playerEntityID common.EntityID) {
	npcs := player.Area.GetNPCs(h.World)

	// Check all NPCs in the area for quests
	for _, npc := range npcs {
		// Find quests offered by this NPC
		for _, questDef := range QuestRegistry {
			if questDef.NPCID != npc.TemplateID {
				continue // This NPC doesn't offer this quest
			}

			if questDef.Dialogue == nil {
				continue // No dialogue configured
			}

			// Process the quest dialogue
			if h.processQuestDialogue(player, playerEntityID, keyword, questDef, npc) {
				return // Dialogue was handled
			}
		}
	}
}

// processQuestDialogue handles dialogue for a specific quest
func (h *QuestDialogueHandler) processQuestDialogue(player *Player, playerEntityID common.EntityID, keyword string, questDef *Quest, npc *NPC) bool {
	// Get or create player's quest component
	questsComp, err := h.World.GetComponent(playerEntityID, "PlayerQuests")
	var quests *PlayerQuests
	if err != nil {
		quests = NewPlayerQuests()
		playerEntities, _ := h.World.FindEntitiesByComponentPredicate("Player", func(i any) bool {
			p, ok := i.(*Player)
			return ok && p == player
		})
		if len(playerEntities) > 0 {
			entity := playerEntities[0]
			h.World.AddComponentToEntity(entity, quests)
		}
	} else {
		quests = questsComp.(*PlayerQuests)
	}

	// Get player quest status
	questID := questDef.ID
	playerQuest := quests.GetOrCreateQuest(questID)

	// Find matching dialogue node
	for _, node := range questDef.Dialogue.Nodes {
		// Check if keyword matches any triggers
		keywordMatches := false
		for _, trigger := range node.Triggers {
			if trigger == keyword {
				keywordMatches = true
				break
			}
		}

		if !keywordMatches {
			continue
		}

		// Check condition
		if node.Condition != nil && !node.Condition(player, playerQuest) {
			continue
		}

		// Show text if available
		if node.Text != "" {
			player.Broadcast(fmt.Sprintf("%s says: %s", npc.Name, node.Text))
		}

		// Execute action if available
		if node.Action != nil {
			h.executeAction(node.Action, player, playerEntityID, playerQuest, questDef, npc, quests)
		}

		return true
	}

	return false
}

// executeAction executes a quest action
func (h *QuestDialogueHandler) executeAction(action *QuestAction, player *Player, playerEntityID common.EntityID, playerQuest *PlayerQuest, questDef *Quest, npc *NPC, quests *PlayerQuests) {
	switch action.Type {
	case ActionCheckProgress:
		h.checkQuestProgress(player, playerEntityID, playerQuest, questDef, npc)

	case ActionAcceptQuest:
		h.acceptQuest(player, questDef, npc, quests)

	case ActionCompleteQuest:
		h.completeQuest(player, playerEntityID, questDef, npc, quests)

	case ActionCustom:
		if action.CustomHandler != nil {
			action.CustomHandler(h, player, playerEntityID, playerQuest, questDef, npc, quests)
		}

	default:
		log.Warn().Msgf("Unknown quest action type: %s", action.Type)
	}
}

// acceptQuest is a generic handler for accepting any quest
func (h *QuestDialogueHandler) acceptQuest(player *Player, questDef *Quest, npc *NPC, quests *PlayerQuests) {
	quests.AddQuest(questDef.ID)
	player.Broadcast(fmt.Sprintf("Quest accepted: %s", questDef.Name))
	player.Broadcast(fmt.Sprintf("%s says: Excellent! Return when you have what I need.", npc.Name))
}

// checkQuestProgress is a generic handler for checking quest progress
func (h *QuestDialogueHandler) checkQuestProgress(player *Player, playerEntityID common.EntityID, playerQuest *PlayerQuest, questDef *Quest, npc *NPC) {
	inventoryComp, err := h.World.GetComponent(playerEntityID, "Inventory")
	if err != nil {
		player.Broadcast("You don't have an inventory.")
		return
	}

	inventory := inventoryComp.(*Inventory)
	inventory.RLock()

	// Check all requirements
	allRequirementsMet := true
	requirementStatus := make(map[string]int) // itemID -> current count

	for _, req := range questDef.Requirements {
		count := 0
		for _, item := range inventory.Items {
			if item.ID == req.ItemID {
				count += item.Quantity
			}
		}
		requirementStatus[req.ItemID] = count
		if count < req.Quantity {
			allRequirementsMet = false
		}
	}
	inventory.RUnlock()

	// Show appropriate message based on quest status
	switch playerQuest.Status {
	case QuestStatusNotStarted:
		// Show what's needed
		for _, req := range questDef.Requirements {
			current := requirementStatus[req.ItemID]
			itemName := getItemName(req.ItemID)
			player.Broadcast(fmt.Sprintf("%s says: I need %d %s. You currently have %d.", npc.Name, req.Quantity, itemName, current))
		}
		player.Broadcast(fmt.Sprintf("%s says: If you're ready to help, just say the word [deal].", npc.Name))

	case QuestStatusInProgress:
		if allRequirementsMet {
			player.Broadcast(fmt.Sprintf("%s says: Excellent! You have everything I need. Say [reward] when you're ready to trade.", npc.Name))
		} else {
			// Show progress
			for _, req := range questDef.Requirements {
				current := requirementStatus[req.ItemID]
				itemName := getItemName(req.ItemID)
				player.Broadcast(fmt.Sprintf("%s says: You have %d/%d %s.", npc.Name, current, req.Quantity, itemName))
			}
		}

	case QuestStatusCompleted:
		player.Broadcast(fmt.Sprintf("%s says: You've already helped me with this matter. Thank you again!", npc.Name))
	}
}

// completeQuest is a generic handler for completing any quest
func (h *QuestDialogueHandler) completeQuest(player *Player, playerEntityID common.EntityID, questDef *Quest, npc *NPC, quests *PlayerQuests) {
	inventoryComp, err := h.World.GetComponent(playerEntityID, "Inventory")
	if err != nil {
		player.Broadcast("You don't have an inventory.")
		return
	}

	inventory := inventoryComp.(*Inventory)

	// Check if player has all required items
	inventory.RLock()
	hasAllItems := true
	for _, req := range questDef.Requirements {
		count := 0
		for _, item := range inventory.Items {
			if item.ID == req.ItemID {
				count += item.Quantity
			}
		}
		if count < req.Quantity {
			hasAllItems = false
			break
		}
	}
	inventory.RUnlock()

	if !hasAllItems {
		player.Broadcast(fmt.Sprintf("%s says: You don't have everything I need yet. Come back when you do.", npc.Name))
		return
	}

	// Remove required items
	for _, req := range questDef.Requirements {
		inventory.RemoveItem(req.ItemID, req.Quantity)
	}

	// Give rewards
	for _, reward := range questDef.Rewards {
		template, exists := ItemTemplates[reward.ItemID]
		if !exists {
			log.Error().Msgf("Item template %s not found", reward.ItemID)
			continue
		}

		item := &Item{
			ID:          template.ID,
			Name:        template.Name,
			Description: template.Description,
			Type:        template.Type,
			Value:       template.Value,
			Stackable:   template.Stackable,
			Quantity:    reward.Quantity,
		}
		inventory.AddItem(item)
		player.Broadcast(fmt.Sprintf("You received: %s x%d", item.Name, reward.Quantity))
	}

	// Mark quest as completed
	quests.CompleteQuest(questDef.ID)

	player.Broadcast(fmt.Sprintf("%s says: Excellent work! Here's your reward. Safe travels!", npc.Name))
}

// getItemName returns a friendly name for an item ID
func getItemName(itemID string) string {
	if template, exists := ItemTemplates[itemID]; exists {
		// Pluralize if needed
		name := template.Name
		if template.Stackable {
			// Simple pluralization - just add 's' for now
			name = name + "s"
		}
		return name
	}
	return itemID
}
