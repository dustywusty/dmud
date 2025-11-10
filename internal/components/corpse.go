package components

import (
	"dmud/internal/common"
	"sync"
	"time"
)

// Corpse represents the remains of a dead entity
type Corpse struct {
	sync.RWMutex

	VictimName string

	VictimID common.EntityID

	// Whether this was a player or NPC
	WasPlayer bool

	// When the corpse was created
	TimeOfDeath time.Time

	// How long before the corpse decays (in seconds)
	DecayTime time.Duration

	Area *Area

	// Future: Items that can be looted
	// Loot []Item
}

// IsDecayed checks if the corpse should be removed
func (c *Corpse) IsDecayed() bool {
	c.RLock()
	defer c.RUnlock()
	return time.Since(c.TimeOfDeath) >= c.DecayTime
}

// GetDescription returns a description of the corpse
func (c *Corpse) GetDescription() string {
	c.RLock()
	defer c.RUnlock()

	if c.WasPlayer {
		return "the corpse of " + c.VictimName
	}
	return "the corpse of " + c.VictimName
}

func NewCorpse(victimName string, victimID common.EntityID, wasPlayer bool, area *Area) *Corpse {
	return &Corpse{
		VictimName:  victimName,
		VictimID:    victimID,
		WasPlayer:   wasPlayer,
		TimeOfDeath: time.Now(),
		DecayTime:   5 * time.Minute, // Corpses last 5 minutes by default
		Area:        area,
	}
}
