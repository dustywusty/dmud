package components

import "sync"

// FactionRank is a reputation tier within a faction. MinRep is the inclusive
// reputation threshold at which a player earns the rank's Title.
type FactionRank struct {
	MinRep int
	Title  string
}

// FactionDef describes a faction and its reputation tiers. Ranks must be listed
// in ascending MinRep order, with the first rank starting at 0.
type FactionDef struct {
	ID    string
	Name  string
	Ranks []FactionRank
}

// RankTitle returns the title a player holds at the given reputation.
func (d *FactionDef) RankTitle(rep int) string {
	title := ""
	for _, r := range d.Ranks {
		if rep >= r.MinRep {
			title = r.Title
		} else {
			break
		}
	}
	if title == "" && len(d.Ranks) > 0 {
		title = d.Ranks[0].Title
	}
	return title
}

// NextRank returns the next rank above the given reputation. ok is false when
// the player is already at (or above) the highest rank.
func (d *FactionDef) NextRank(rep int) (rank FactionRank, ok bool) {
	for _, r := range d.Ranks {
		if rep < r.MinRep {
			return r, true
		}
	}
	return FactionRank{}, false
}

// FactionRegistry holds all factions in the game, keyed by faction ID.
var FactionRegistry = map[string]*FactionDef{
	"cinderhollow": {
		ID:   "cinderhollow",
		Name: "the Cinderhollow Goblins",
		Ranks: []FactionRank{
			{MinRep: 0, Title: "Stranger"},
			{MinRep: 15, Title: "Acquaintance"},
			{MinRep: 40, Title: "Friend"},
			{MinRep: 80, Title: "Honored"},
			{MinRep: 150, Title: "Beloved"},
		},
	},
}

// Factions is a per-player component tracking reputation with each faction.
type Factions struct {
	sync.RWMutex
	Reputation map[string]int // factionID -> reputation points
}

func NewFactions() *Factions {
	return &Factions{Reputation: make(map[string]int)}
}

func (f *Factions) Type() string { return "Factions" }

// Get returns the player's reputation with a faction (0 if none).
func (f *Factions) Get(factionID string) int {
	f.RLock()
	defer f.RUnlock()
	return f.Reputation[factionID]
}

// Add increases reputation by amount (clamped at zero) and returns the new total.
func (f *Factions) Add(factionID string, amount int) int {
	f.Lock()
	defer f.Unlock()
	if f.Reputation == nil {
		f.Reputation = make(map[string]int)
	}
	f.Reputation[factionID] += amount
	if f.Reputation[factionID] < 0 {
		f.Reputation[factionID] = 0
	}
	return f.Reputation[factionID]
}

// Set overwrites reputation with a faction (used when restoring saved state).
func (f *Factions) Set(factionID string, amount int) {
	f.Lock()
	defer f.Unlock()
	if f.Reputation == nil {
		f.Reputation = make(map[string]int)
	}
	f.Reputation[factionID] = amount
}

// Snapshot returns a copy of the reputation map, safe to read without locking.
func (f *Factions) Snapshot() map[string]int {
	f.RLock()
	defer f.RUnlock()
	out := make(map[string]int, len(f.Reputation))
	for k, v := range f.Reputation {
		out[k] = v
	}
	return out
}
