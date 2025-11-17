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

	// Items that can be looted from the corpse
	Inventory *Inventory

	// When the corpse was fully looted (empty)
	LootedAt *time.Time
}

// IsDecayed checks if the corpse should be removed
func (c *Corpse) IsDecayed() bool {
	c.RLock()
	defer c.RUnlock()

	// If corpse was fully looted, decay after 5 seconds
	if c.LootedAt != nil {
		return time.Since(*c.LootedAt) >= 5*time.Second
	}

	// Otherwise use normal decay time
	return time.Since(c.TimeOfDeath) >= c.DecayTime
}

// MarkAsLooted marks the corpse as having been fully looted
func (c *Corpse) MarkAsLooted() {
	c.Lock()
	defer c.Unlock()
	now := time.Now()
	c.LootedAt = &now
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

func NewCorpse(victimName string, victimID common.EntityID, wasPlayer bool, area *Area, inventory *Inventory) *Corpse {
	return &Corpse{
		VictimName:  victimName,
		VictimID:    victimID,
		WasPlayer:   wasPlayer,
		TimeOfDeath: time.Now(),
		DecayTime:   30 * time.Minute,
		Area:        area,
		Inventory:   inventory,
	}
}
