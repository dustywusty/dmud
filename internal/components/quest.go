package components

import "sync"

type QuestStatus int

const (
	QuestStatusNotStarted QuestStatus = iota
	QuestStatusInProgress
	QuestStatusCompleted
)

type QuestRequirement struct {
	ItemID   string
	Quantity int
}

type QuestReward struct {
	ItemID   string
	Quantity int
}

type Quest struct {
	ID           string
	Name         string
	Description  string
	Requirements []QuestRequirement
	Rewards      []QuestReward
	NPCID        string // Which NPC offers this quest
}

// PlayerQuest tracks a player's progress on a quest
type PlayerQuest struct {
	sync.RWMutex
	QuestID string
	Status  QuestStatus
}

// PlayerQuests component tracks all quests for a player
type PlayerQuests struct {
	sync.RWMutex
	Quests map[string]*PlayerQuest // QuestID -> PlayerQuest
}

func NewPlayerQuests() *PlayerQuests {
	return &PlayerQuests{
		Quests: make(map[string]*PlayerQuest),
	}
}

func (pq *PlayerQuests) Type() string {
	return "PlayerQuests"
}

func (pq *PlayerQuests) HasQuest(questID string) bool {
	pq.RLock()
	defer pq.RUnlock()
	_, exists := pq.Quests[questID]
	return exists
}

func (pq *PlayerQuests) GetQuestStatus(questID string) QuestStatus {
	pq.RLock()
	defer pq.RUnlock()
	if quest, exists := pq.Quests[questID]; exists {
		quest.RLock()
		defer quest.RUnlock()
		return quest.Status
	}
	return QuestStatusNotStarted
}

func (pq *PlayerQuests) AddQuest(questID string) {
	pq.Lock()
	defer pq.Unlock()
	pq.Quests[questID] = &PlayerQuest{
		QuestID: questID,
		Status:  QuestStatusInProgress,
	}
}

func (pq *PlayerQuests) CompleteQuest(questID string) {
	pq.Lock()
	defer pq.Unlock()
	if quest, exists := pq.Quests[questID]; exists {
		quest.Lock()
		quest.Status = QuestStatusCompleted
		quest.Unlock()
	}
}

// QuestRegistry holds all available quests
var QuestRegistry = map[string]Quest{
	"goblin_ears": {
		ID:          "goblin_ears",
		Name:        "Goblin Menace",
		Description: "The traveling merchant needs help dealing with goblins. Bring him 10 goblin ears as proof of your deeds.",
		Requirements: []QuestRequirement{
			{ItemID: "goblin_ear", Quantity: 10},
		},
		Rewards: []QuestReward{
			{ItemID: "leather_helmet", Quantity: 1},
			{ItemID: "leather_chest", Quantity: 1},
			{ItemID: "leather_legs", Quantity: 1},
			{ItemID: "leather_boots", Quantity: 1},
			{ItemID: "gold_coin", Quantity: 50},
		},
		NPCID: "merchant",
	},
}
