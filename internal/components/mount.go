package components

import "time"

// MountStats are the fixed attributes a horse item confers when ridden. They are
// keyed by item ID in MountRegistry, mirroring how vendors and factions are
// defined in code while the items themselves live in ItemTemplates.
type MountStats struct {
	ItemID       string
	Name         string
	Speed        int // rooms covered per `ride <direction>` (move speed)
	MaxEndurance int // gallop-rooms available before the horse must rest
}

// MountRegistry holds every rideable horse, keyed by item ID.
var MountRegistry = map[string]*MountStats{
	"pony":    {ItemID: "pony", Name: "Dapple Pony", Speed: 2, MaxEndurance: 6},
	"courser": {ItemID: "courser", Name: "Midnight Courser", Speed: 3, MaxEndurance: 10},
}

func MountStatsFor(itemID string) (*MountStats, bool) {
	m, ok := MountRegistry[itemID]
	return m, ok
}

// IsMountItem reports whether an item ID is a rideable horse.
func IsMountItem(itemID string) bool {
	_, ok := MountRegistry[itemID]
	return ok
}

// Mount is the horse a player is currently riding. Like NPC it carries no mutex:
// it is only ever read or written on the game-loop goroutine (command handlers),
// so the actor model already serializes access.
type Mount struct {
	ItemID       string
	Name         string
	Speed        int
	Endurance    int
	MaxEndurance int
	LastRest     time.Time
}

func (m *Mount) Type() string { return "Mount" }

const mountRegenInterval = 5 * time.Second

// Regen lazily restores endurance based on the time elapsed since the last
// update, so mounts need no dedicated system tick. Call it before reading
// Endurance.
func (m *Mount) Regen() {
	if m.MaxEndurance <= 0 {
		return
	}
	if m.LastRest.IsZero() || m.Endurance >= m.MaxEndurance {
		m.LastRest = time.Now()
		if m.Endurance > m.MaxEndurance {
			m.Endurance = m.MaxEndurance
		}
		return
	}
	gained := int(time.Since(m.LastRest) / mountRegenInterval)
	if gained <= 0 {
		return
	}
	m.Endurance += gained
	if m.Endurance > m.MaxEndurance {
		m.Endurance = m.MaxEndurance
	}
	m.LastRest = m.LastRest.Add(time.Duration(gained) * mountRegenInterval)
}
