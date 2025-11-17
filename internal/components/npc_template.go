package components

import (
	"math/rand"
	"time"
)

type NPCBehavior int

const (
    BehaviorPassive NPCBehavior = iota
    BehaviorAggressive
    BehaviorDefensive
    BehaviorFriendly
    BehaviorMerchant
    BehaviorGuard
)

type LootDrop struct {
	ItemID   string
	Chance   float64 // 0.0 to 1.0
	MinCount int
	MaxCount int
}

type NPCTemplate struct {
	ID          string
	Name        string
	Description string
	Health      int
	MinDamage   int
	MaxDamage   int
	Behavior    NPCBehavior
	Dialogue    []string // Random things they might say
	RespawnTime time.Duration
	Stationary  bool       // If true, NPC will not wander between areas
	LootTable   []LootDrop // Possible items this NPC can drop
}

var NPCTemplates = map[string]NPCTemplate{
	"rat": {
		ID:          "rat",
		Name:        "a small rat",
		Description: "A small, scurrying rat with beady eyes and a twitching nose.",
		Health:      20,
		MinDamage:   1,
		MaxDamage:   5,
		Behavior:    BehaviorPassive,
		Dialogue:    []string{"*squeaks*", "*scurries around*"},
		RespawnTime: 30 * time.Second,
		LootTable: []LootDrop{
			{ItemID: "rat_fur", Chance: 0.5, MinCount: 1, MaxCount: 2},
			{ItemID: "rat_tail", Chance: 0.3, MinCount: 1, MaxCount: 1},
		},
	},
	"goblin": {
		ID:          "goblin",
		Name:        "a sneaky goblin",
		Description: "A small, green-skinned goblin with sharp teeth and cunning eyes.",
		Health:      50,
		MinDamage:   5,
		MaxDamage:   15,
		Behavior:    BehaviorAggressive,
		Dialogue:    []string{"Grrr!", "Me smash you!", "Give shinies!"},
		RespawnTime: 60 * time.Second,
		LootTable: []LootDrop{
			{ItemID: "goblin_ear", Chance: 0.7, MinCount: 1, MaxCount: 2},
			{ItemID: "rusty_dagger", Chance: 0.4, MinCount: 1, MaxCount: 1},
			{ItemID: "gold_coin", Chance: 0.9, MinCount: 1, MaxCount: 5},
		},
	},
	"guard": {
		ID:          "guard",
		Name:        "a town guard",
		Description: "A stern-looking guard in chainmail armor, watching for trouble.",
		Health:      150,
		MinDamage:   10,
		MaxDamage:   25,
		Behavior:    BehaviorGuard,
		Dialogue:    []string{"Move along, citizen.", "No trouble here!", "Keep the peace."},
		RespawnTime: 120 * time.Second,
		LootTable: []LootDrop{
			{ItemID: "gold_coin", Chance: 1.0, MinCount: 5, MaxCount: 15},
		},
	},
	"merchant": {
		ID:          "merchant",
		Name:        "a traveling merchant",
		Description: "A portly merchant with a warm smile and keen eyes for business.",
		Health:      80,
		MinDamage:   5,
		MaxDamage:   10,
		Behavior:    BehaviorMerchant,
		Dialogue:    []string{"Fine wares for sale!", "Come, see my goods!", "Best prices in town!"},
		RespawnTime: 180 * time.Second,
		LootTable: []LootDrop{
			{ItemID: "gold_coin", Chance: 1.0, MinCount: 10, MaxCount: 30},
		},
	},
	"chicken": {
		ID:          "chicken",
		Name:        "a chicken",
		Description: "A plump chicken pecking at the ground, oblivious to its surroundings.",
		Health:      20,
		MinDamage:   1,
		MaxDamage:   2,
		Behavior:    BehaviorPassive,
		Dialogue:    []string{"*cluck cluck*", "*bawk!*"},
		RespawnTime: 15 * time.Second,
		Stationary:  true,
		LootTable: []LootDrop{
			{ItemID: "chicken_feather", Chance: 0.8, MinCount: 1, MaxCount: 3},
			{ItemID: "raw_chicken", Chance: 1.0, MinCount: 1, MaxCount: 1},
		},
	},
}

// GenerateLoot creates inventory items based on the NPC's loot table
func GenerateLoot(templateID string) *Inventory {
	template, exists := NPCTemplates[templateID]
	if !exists {
		return NewInventory(0)
	}

	inventory := NewInventory(0) // NPCs have unlimited inventory

	for _, lootDrop := range template.LootTable {
		// Roll for chance
		if rand.Float64() <= lootDrop.Chance {
			// Determine quantity
			count := lootDrop.MinCount
			if lootDrop.MaxCount > lootDrop.MinCount {
				count = lootDrop.MinCount + rand.Intn(lootDrop.MaxCount-lootDrop.MinCount+1)
			}

			// Create and add item
			item := CreateItem(lootDrop.ItemID, count)
			if item != nil {
				inventory.AddItem(item)
			}
		}
	}

	return inventory
}
