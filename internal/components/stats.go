package components

import (
	"fmt"
	"math/rand"
)

// StatType enumerates the character attributes. Order is stable (used for the
// fixed-size Values array and persistence keys).
type StatType int

const (
	STR StatType = iota // Strength    — melee damage
	DEX                 // Dexterity   — endurance efficiency
	CON                 // Constitution— max HP
	INT                 // Intelligence— arcane (fire) damage
	WIS                 // Wisdom      — healing
	numStats
)

const (
	statMin             = 1   // hard floor (racial penalties can dip below the base)
	statBase            = 10  // a Human's starting value
	statCap             = 100 // and the ceiling
	statTrainBaseChance = 18  // % chance to gain at the base value, scaling to 0 at the cap
)

// StatBase is the starting value of a stat, exported so other packages can scale
// effects relative to "untrained" (e.g. buff duration grows with skill above it).
const StatBase = statBase

var statNames = [numStats]string{"Strength", "Dexterity", "Constitution", "Intelligence", "Wisdom"}
var statAbbrev = [numStats]string{"STR", "DEX", "CON", "INT", "WIS"}

// Stats holds use-trained attributes plus the creature's race and size. Players
// train the values; NPCs get fixed race-derived values. Like Health it lives
// only on the game-loop goroutine, so it carries no mutex.
type Stats struct {
	Values [numStats]int
	Race   string
	Size   Size
	Class  string // class key (e.g. "druid"); "" = classless
}

func (s *Stats) Type() string { return "Stats" }

// NewStats makes a baseline Human.
func NewStats() *Stats { return NewStatsForRace("human") }

// NewStatsForRace makes stats seeded from a race's offsets and size. Unknown
// races fall back to a balanced Human.
func NewStatsForRace(raceKey string) *Stats {
	s := &Stats{Race: "Human", Size: SizeMedium}
	for i := range s.Values {
		s.Values[i] = statBase
	}
	if def, ok := RaceFor(raceKey); ok {
		s.Race = def.Name
		s.Size = def.Size
		for i := StatType(0); i < numStats; i++ {
			s.Set(i, statBase+def.StatMods[i])
		}
	}
	return s
}

func (s *Stats) Get(t StatType) int {
	if int(t) < 0 || int(t) >= int(numStats) {
		return statBase
	}
	return s.Values[t]
}

// Set clamps and stores a stat (used when restoring from persistence).
func (s *Stats) Set(t StatType, v int) {
	if int(t) < 0 || int(t) >= int(numStats) {
		return
	}
	if v < statMin {
		v = statMin
	}
	if v > statCap {
		v = statCap
	}
	s.Values[t] = v
}

func StatName(t StatType) string {
	if int(t) < 0 || int(t) >= int(numStats) {
		return "?"
	}
	return statNames[t]
}

func StatAbbrev(t StatType) string {
	if int(t) < 0 || int(t) >= int(numStats) {
		return "?"
	}
	return statAbbrev[t]
}

// AllStats returns every stat type in display order.
func AllStats() []StatType {
	out := make([]StatType, 0, numStats)
	for i := StatType(0); i < numStats; i++ {
		out = append(out, i)
	}
	return out
}

// Train rolls a use-based increase for a stat: the chance is highest at the base
// value and falls to zero at the cap, so growth slows the stronger you get.
// Returns whether the stat rose and its new value.
func (s *Stats) Train(t StatType) (raised bool, newVal int) {
	cur := s.Get(t)
	if cur >= statCap {
		return false, cur
	}
	chance := statTrainBaseChance * (statCap - cur) / (statCap - statBase)
	if rand.Intn(100) < chance {
		s.Set(t, cur+1)
		return true, cur + 1
	}
	return false, cur
}

// --- modifiers: how stats bend the actions that train them ---

// statFactor is a gentle multiplier: 1.0 at base, +1% per point above it.
func statFactor(v int) float64 { return 1 + float64(v-statBase)*0.01 }

// Factor is the generic +1%/point-above-base multiplier for any stat.
func (s *Stats) Factor(t StatType) float64 { return statFactor(s.Get(t)) }

func (s *Stats) MeleeFactor() float64  { return statFactor(s.Get(STR)) }
func (s *Stats) ArcaneFactor() float64 { return statFactor(s.Get(INT)) }
func (s *Stats) HealFactor() float64   { return statFactor(s.Get(WIS)) }

// HPBonus is the extra maximum health Constitution grants (+3 per point).
func (s *Stats) HPBonus() int {
	if v := s.Get(CON) - statBase; v > 0 {
		return v * 3
	}
	return 0
}

// CostFactor scales an endurance cost down with Dexterity, to a 50% floor.
func (s *Stats) CostFactor() float64 {
	red := float64(s.Get(DEX)-statBase) * 0.004
	if red < 0 {
		red = 0
	}
	if red > 0.5 {
		red = 0.5
	}
	return 1 - red
}

// TrainStat trains a player's stat from an action and announces any increase.
// Safe to call with a nil Stats (no-op), so callers needn't guard.
func TrainStat(p *Player, s *Stats, t StatType) {
	if p == nil || s == nil {
		return
	}
	if raised, val := s.Train(t); raised {
		p.Broadcast(fmt.Sprintf("Your %s feels sharper. (%s %d)", StatName(t), StatAbbrev(t), val))
	}
}
