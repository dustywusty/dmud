// internal/components/spawn.go
package components

import (
	"dmud/internal/common"
	"sync"
	"time"
)

type SpawnType int

const (
    SpawnTypeNPC SpawnType = iota
    SpawnTypeItem
    SpawnTypeResource
)

type SpawnConfig struct {
    Type        SpawnType
    TemplateID  string      // ID of the NPC template to spawn
    MinCount    int         // Minimum number to maintain
    MaxCount    int         // Maximum number allowed
    RespawnTime time.Duration // Time before respawning
    Chance      float64     // Spawn chance (0.0 to 1.0)
}

type Spawn struct {
    sync.RWMutex

    Configs      []SpawnConfig
    ActiveSpawns map[string]common.EntityID // templateID -> entityIDs
    LastSpawn    time.Time
    RoomID       common.EntityID
}

func NewSpawn(roomID common.EntityID) *Spawn {
    return &Spawn{
        Configs:      make([]SpawnConfig, 0),
        ActiveSpawns: make(map[string]common.EntityID),
        RoomID:       roomID,
        LastSpawn:    time.Now(),
    }
}
