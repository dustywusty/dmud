package components

import (
	"dmud/internal/common"
	"sync"
	"time"
)

type Corpse struct {
	sync.RWMutex

	VictimName  string
	VictimID    common.EntityID
	WasPlayer   bool
	TimeOfDeath time.Time
	DecayTime   time.Duration // How long before the corpse decays (in seconds)
	Area        *Area
	Inventory   *Inventory
	LootedAt    *time.Time
}

func (c *Corpse) IsDecayed() bool {
	c.RLock()
	defer c.RUnlock()
	if c.LootedAt != nil { // If corpse was fully looted, decay after 5 seconds
		return time.Since(*c.LootedAt) >= 5*time.Second
	} // Otherwise use normal decay time
	return time.Since(c.TimeOfDeath) >= c.DecayTime
}

func (c *Corpse) MarkAsLooted() {
	c.Lock()
	defer c.Unlock()
	now := time.Now()
	c.LootedAt = &now
}

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
