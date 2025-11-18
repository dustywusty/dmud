package components

import "dmud/internal/common"

// QuestActionType defines the type of action a dialogue node performs
type QuestActionType string

const (
	ActionCheckProgress QuestActionType = "check_progress" // Show quest progress
	ActionAcceptQuest   QuestActionType = "accept_quest"   // Accept the quest
	ActionCompleteQuest QuestActionType = "complete_quest" // Turn in the quest for rewards
	ActionCustom        QuestActionType = "custom"         // Custom action defined by quest
)

// QuestAction defines what happens when a dialogue node is triggered
type QuestAction struct {
	Type QuestActionType

	// For custom actions, this function will be called
	// It receives the handler so it can access World and other methods
	CustomHandler func(handler *QuestDialogueHandler, player *Player, playerEntityID common.EntityID, playerQuest *PlayerQuest, questDef *Quest, npc *NPC, quests *PlayerQuests)
}
