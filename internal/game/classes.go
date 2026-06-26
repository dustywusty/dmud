package game

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/util"
)

// classAffinityBonus is the one-time boost to a class's primary stat the first
// time a character takes any class.
const classAffinityBonus = 5

// ClassDef is a character class: a thematic bundle of spell schools plus the
// attribute that powers it. Classes don't gate gameplay yet — this registry is
// the source of truth the in-game `classes` command and the generated docs read.
type ClassDef struct {
	Key         string
	Name        string
	Description string
	Schools     []string // spell schools this class draws from
	Abilities   []string // non-spell signature abilities (e.g. "shapeshift")
	PrimaryStat components.StatType
}

// classRegistry is an ordered list so listings and generated docs are stable.
var classRegistry = []*ClassDef{
	{
		Key: "pyromancer", Name: "Pyromancer",
		Description: "A glass cannon of raw arcane fire — devastating at range, fragile up close.",
		Schools:     []string{"pyromancy"}, PrimaryStat: components.INT,
	},
	{
		Key: "cleric", Name: "Cleric",
		Description: "A battle-priest: mends allies, smites foes with holy light, and shields with blessings.",
		Schools:     []string{"restoration", "holy"}, PrimaryStat: components.WIS,
	},
	{
		Key: "warlock", Name: "Warlock",
		Description: "A pact-bound dominator who bends minds to their will and drains the life from those who resist.",
		Schools:     []string{"domination", "shadow"}, PrimaryStat: components.INT,
	},
	{
		Key: "druid", Name: "Druid",
		Description: "A keeper of the wild — mends wounds, calls down moonlight, and takes beast shape to rend foes in melee.",
		Schools:     []string{"restoration", "lunar"},
		Abilities:   []string{"shapeshift"},
		PrimaryStat: components.WIS,
	},
}

// classByKey looks up a class by its (case-insensitive) key.
func classByKey(key string) (*ClassDef, bool) {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, c := range classRegistry {
		if c.Key == key {
			return c, true
		}
	}
	return nil, false
}

// spellsInSchool returns the (deduped) spells of a school, ordered by level.
func spellsInSchool(school string) []*SpellDefinition {
	seen := make(map[*SpellDefinition]bool)
	var out []*SpellDefinition
	for _, def := range spellRegistry {
		if def.School != school || seen[def] {
			continue
		}
		seen[def] = true
		out = append(out, def)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].MinLevel != out[j].MinLevel {
			return out[i].MinLevel < out[j].MinLevel
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// spellNamesInSchools lists the spell names a set of schools grants, in order.
func spellNamesInSchools(schools []string) []string {
	var names []string
	for _, school := range schools {
		for _, def := range spellsInSchool(school) {
			names = append(names, def.Name)
		}
	}
	return names
}

// classKeys returns the class keys in registry order, for prompts.
func classKeys() []string {
	keys := make([]string, 0, len(classRegistry))
	for _, c := range classRegistry {
		keys = append(keys, c.Key)
	}
	return keys
}

// classAllowsSchool reports whether a class may cast a school's spells. Classless
// characters (no class chosen) are unrestricted.
func classAllowsSchool(stats *components.Stats, school string) bool {
	if stats == nil || stats.Class == "" {
		return true
	}
	def, ok := classByKey(stats.Class)
	if !ok {
		return true
	}
	for _, s := range def.Schools {
		if s == school {
			return true
		}
	}
	return false
}

// classAllowsAbility reports whether a class has a signature ability (e.g.
// "shapeshift"). Classless characters are unrestricted.
func classAllowsAbility(stats *components.Stats, ability string) bool {
	if stats == nil || stats.Class == "" {
		return true
	}
	def, ok := classByKey(stats.Class)
	if !ok {
		return true
	}
	for _, a := range def.Abilities {
		if a == ability {
			return true
		}
	}
	return false
}

// macroBinding / macroSet are the EVENT|{macros} push: the server's suggested
// hotkey loadout, applied by the client to its macro bar.
type macroBinding struct {
	Slot  int    `json:"slot"`
	Label string `json:"label"`
	Cmd   string `json:"cmd"`
}

type macroSet struct {
	Type   string         `json:"type"`
	Source string         `json:"source,omitempty"`
	Set    []macroBinding `json:"set"`
}

// pushClassMacros sends the client a hotkey loadout for the class's currently
// castable spells (one per slot), so picking a class wires up its bar.
func (g *Game) pushClassMacros(player *components.Player, entityID common.EntityID, def *ClassDef) {
	level := g.casterLevel(entityID)
	var set []macroBinding
	slot := 1
	for _, school := range def.Schools {
		for _, sp := range spellsInSchool(school) {
			if sp.MinLevel > level || slot > 10 {
				continue
			}
			set = append(set, macroBinding{Slot: slot, Label: sp.Name, Cmd: "cast " + sp.Name})
			slot++
		}
	}
	if len(set) == 0 {
		return
	}
	if data, err := json.Marshal(macroSet{Type: "macros", Source: "class:" + def.Key, Set: set}); err == nil {
		player.Broadcast(util.TagMessage("EVENT", string(data)))
	}
	player.Broadcast(fmt.Sprintf("Wired %d hotkeys for your %s spells -- check your bar.", len(set), def.Name))
}

// handleClass shows or sets the player's class. Picking grants a one-time
// affinity boost to the primary stat, restricts casting to the class's schools,
// and pushes a macro loadout to the client.
func (g *Game) handleClass(player *components.Player, args []string, game *Game) {
	entityID, err := g.getPlayerEntity(player)
	if err != nil {
		player.Broadcast("Unable to read your character.")
		return
	}
	stats := g.getStats(entityID)
	if stats == nil {
		player.Broadcast("You have no stats.")
		return
	}

	if len(args) == 0 {
		if def, ok := classByKey(stats.Class); ok {
			player.Broadcast(fmt.Sprintf("You walk the path of the %s.", def.Name))
		} else {
			player.Broadcast("You have no class yet.")
		}
		player.Broadcast("Choose with: class <" + strings.Join(classKeys(), "|") + ">   ('classes' for details)")
		return
	}

	def, ok := classByKey(args[0])
	if !ok {
		player.Broadcast("No such class. Choose: " + strings.Join(classKeys(), ", "))
		return
	}

	firstClass := stats.Class == ""
	stats.Class = def.Key
	if firstClass {
		stats.Set(def.PrimaryStat, stats.Get(def.PrimaryStat)+classAffinityBonus)
		player.Broadcast(fmt.Sprintf("You take up the path of the %s. (%s affinity +%d)",
			def.Name, components.StatAbbrev(def.PrimaryStat), classAffinityBonus))
	} else {
		player.Broadcast(fmt.Sprintf("You retrain as a %s.", def.Name))
	}

	g.pushClassMacros(player, entityID, def)
	player.BroadcastState(g.world.AsWorldLike(), entityID)
	g.creationStep(player, "class")
}

// handleClasses lists the available classes, their schools, and their spells.
func (g *Game) handleClasses(player *components.Player, args []string, game *Game) {
	player.Broadcast("== Classes ==")
	for _, c := range classRegistry {
		player.Broadcast(fmt.Sprintf("%s -- %s", c.Name, c.Description))
		player.Broadcast(fmt.Sprintf("  primary %s   schools: %s",
			components.StatAbbrev(c.PrimaryStat), strings.Join(c.Schools, ", ")))
		if spells := spellNamesInSchools(c.Schools); len(spells) > 0 {
			player.Broadcast("  spells: " + strings.Join(spells, ", "))
		}
	}
}
