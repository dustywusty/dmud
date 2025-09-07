package models

import (
	"github.com/rs/zerolog/log"
)

// Character represents a persistent player character.
type Character struct {
	Name string
	X    int
	Y    int
	Z    int
}

// Save persists the character to the database. This is a placeholder
// implementation that just logs the save operation.
func (c *Character) Save() error {
	log.Debug().Msgf("saving character %s at (%d,%d,%d)", c.Name, c.X, c.Y, c.Z)
	return nil
}
