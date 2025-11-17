package components

import "time"

type NPCBehavior int

const (
    BehaviorPassive NPCBehavior = iota
    BehaviorAggressive
    BehaviorDefensive
    BehaviorFriendly
    BehaviorMerchant
    BehaviorGuard
)

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
    Stationary  bool // If true, NPC will not wander between areas
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
    },
}
