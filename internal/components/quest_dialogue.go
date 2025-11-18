package components

// DialogueNode represents a single piece of NPC dialogue with keyword triggers
type DialogueNode struct {
	// The text the NPC says (can contain [bracketed keywords])
	Text string
	// Keywords that trigger this node (e.g., "work", "goblin", "reward")
	Triggers []string
	// Next dialogue node ID to transition to after this (optional)
	NextNode string
	// Condition function to check if this node should be shown
	Condition func(player *Player, quest *PlayerQuest) bool
	// Action to perform when this node is triggered
	Action *QuestAction
}

// QuestDialogue defines the complete dialogue tree for a quest
type QuestDialogue struct {
	// NPC name who gives this quest
	NPCName string
	// Greeting shown when player first hails the NPC
	Greeting string
	// Map of keyword -> dialogue node
	Nodes map[string]*DialogueNode
}
