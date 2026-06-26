package components

import "strings"

// Size is a creature's physical scale. It gates movement through narrow exits
// and is the seed of broader "scale" mechanics later.
type Size int

const (
	SizeSmall  Size = 1
	SizeMedium Size = 2
	SizeLarge  Size = 3
	SizeHuge   Size = 4
)

func (s Size) String() string {
	switch s {
	case SizeSmall:
		return "Small"
	case SizeLarge:
		return "Large"
	case SizeHuge:
		return "Huge"
	default:
		return "Medium"
	}
}

// RaceDef is a playable/monster race: a set of stat offsets from the base value
// plus a physical size. Keyed by lowercase name in RaceRegistry, mirroring how
// vendors/mounts/factions are defined in code.
type RaceDef struct {
	Name     string
	Size     Size
	StatMods [numStats]int // offsets applied to statBase per stat
}

// RaceRegistry holds every race. Index the StatMods with the StatType constants.
var RaceRegistry = map[string]*RaceDef{
	"human": {Name: "Human", Size: SizeMedium, StatMods: [numStats]int{}},
	"ogre": {Name: "Ogre", Size: SizeLarge, StatMods: [numStats]int{
		STR: 10, DEX: -6, CON: 10, INT: -8, WIS: -4,
	}},
	"goblin": {Name: "Goblin", Size: SizeSmall, StatMods: [numStats]int{
		STR: -4, DEX: 6, INT: 2,
	}},
	"elf": {Name: "Elf", Size: SizeMedium, StatMods: [numStats]int{
		STR: -4, DEX: 4, CON: -6, INT: 8, WIS: 4,
	}},
	"dwarf": {Name: "Dwarf", Size: SizeMedium, StatMods: [numStats]int{
		STR: 6, DEX: -4, CON: 8, WIS: 2,
	}},
}

// RaceFor looks up a race by (case-insensitive) name.
func RaceFor(name string) (*RaceDef, bool) {
	def, ok := RaceRegistry[strings.ToLower(strings.TrimSpace(name))]
	return def, ok
}

// RaceNames returns the available race keys (sorted-ish for help text).
func RaceNames() []string {
	return []string{"human", "dwarf", "elf", "goblin", "ogre"}
}
