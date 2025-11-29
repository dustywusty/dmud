package components

import (
	"encoding/json"
	"math/rand"
	"os"
	"time"

	"github.com/rs/zerolog/log"
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

var behaviorFromString = map[string]NPCBehavior{
	"passive":    BehaviorPassive,
	"aggressive": BehaviorAggressive,
	"defensive":  BehaviorDefensive,
	"friendly":   BehaviorFriendly,
	"merchant":   BehaviorMerchant,
	"guard":      BehaviorGuard,
}

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

// JSON structs for loading
type lootDropJSON struct {
	ItemID   string  `json:"item_id"`
	Chance   float64 `json:"chance"`
	MinCount int     `json:"min_count"`
	MaxCount int     `json:"max_count"`
}

type npcTemplateJSON struct {
	ID                 string         `json:"id"`
	Name               string         `json:"name"`
	Description        string         `json:"description"`
	Health             int            `json:"health"`
	MinDamage          int            `json:"min_damage"`
	MaxDamage          int            `json:"max_damage"`
	Behavior           string         `json:"behavior"`
	Dialogue           []string       `json:"dialogue"`
	RespawnTimeSeconds int            `json:"respawn_time_seconds"`
	Stationary         bool           `json:"stationary,omitempty"`
	LootTable          []lootDropJSON `json:"loot_table"`
}

var NPCTemplates = make(map[string]NPCTemplate)

func LoadNPCTemplates(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var templates []npcTemplateJSON
	if err := json.Unmarshal(data, &templates); err != nil {
		return err
	}

	for _, t := range templates {
		lootTable := make([]LootDrop, len(t.LootTable))
		for i, l := range t.LootTable {
			lootTable[i] = LootDrop{
				ItemID:   l.ItemID,
				Chance:   l.Chance,
				MinCount: l.MinCount,
				MaxCount: l.MaxCount,
			}
		}

		behavior, ok := behaviorFromString[t.Behavior]
		if !ok {
			log.Warn().Msgf("Unknown behavior '%s' for NPC '%s', defaulting to passive", t.Behavior, t.ID)
			behavior = BehaviorPassive
		}

		NPCTemplates[t.ID] = NPCTemplate{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Health:      t.Health,
			MinDamage:   t.MinDamage,
			MaxDamage:   t.MaxDamage,
			Behavior:    behavior,
			Dialogue:    t.Dialogue,
			RespawnTime: time.Duration(t.RespawnTimeSeconds) * time.Second,
			Stationary:  t.Stationary,
			LootTable:   lootTable,
		}
	}

	log.Info().Msgf("Loaded %d NPC templates from %s", len(templates), filename)
	return nil
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
