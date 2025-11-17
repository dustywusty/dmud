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
	TemplateID  string        // ID of the NPC template to spawn
	MinCount    int           // Minimum number to maintain
	MaxCount    int           // Maximum number allowed
	RespawnTime time.Duration // Time before respawning
	Chance      float64       // Spawn chance (0.0 to 1.0)
}

type Spawn struct {
	sync.RWMutex

	Configs      []SpawnConfig
	ActiveSpawns map[string][]common.EntityID // templateID -> list of entityIDs
	LastSpawn    time.Time
	AreaID       common.EntityID
}

func NewSpawn(areaID common.EntityID) *Spawn {
	return &Spawn{
		Configs:      make([]SpawnConfig, 0),
		ActiveSpawns: make(map[string][]common.EntityID),
		AreaID:       areaID,
		LastSpawn:    time.Now(),
	}
}
