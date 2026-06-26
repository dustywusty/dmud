package components

import "strings"

// FormDef is a Druid beast form: a melee profile (natural-weapon damage) plus
// flavor, keyed by lowercase name in FormRegistry like races and mounts.
type FormDef struct {
	Key         string
	Name        string // how it reads in the room, e.g. "a great bear"
	Label       string // short tag for the status bar, e.g. "Bear"
	Description string
	MinLevel    int
	MinDamage   int
	MaxDamage   int
}

// FormRegistry holds every shapeshift form.
var FormRegistry = map[string]*FormDef{
	"bear": {
		Key: "bear", Name: "a great bear", Label: "Bear",
		Description: "A hulking brown bear — all muscle, claws, and bad intentions.",
		MinLevel:    1, MinDamage: 14, MaxDamage: 26,
	},
	"wolf": {
		Key: "wolf", Name: "a dire wolf", Label: "Dire Wolf",
		Description: "A lean grey dire wolf with a punishing, worrying bite.",
		MinLevel:    3, MinDamage: 11, MaxDamage: 23,
	},
	"panther": {
		Key: "panther", Name: "a sleek panther", Label: "Panther",
		Description: "A coiled black panther — fast, silent, and lethal.",
		MinLevel:    6, MinDamage: 16, MaxDamage: 34,
	},
}

// FormFor looks up a form by (case-insensitive) key.
func FormFor(key string) (*FormDef, bool) {
	d, ok := FormRegistry[strings.ToLower(strings.TrimSpace(key))]
	return d, ok
}

// FormNames returns the form keys in display order.
func FormNames() []string { return []string{"bear", "wolf", "panther"} }

// Shift marks a player as currently in a beast form. It is transient — players
// revert to their own shape on logout — so it isn't persisted.
type Shift struct {
	Form string // form key
}

func (s *Shift) Type() string { return "Shift" }
