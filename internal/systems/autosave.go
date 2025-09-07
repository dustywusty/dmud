package systems

import (
	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/models"
	"os"
	"strconv"

	"github.com/rs/zerolog/log"
)

// AutosaveSystem periodically persists active player data.
type AutosaveSystem struct {
	counter  int
	interval int
}

// NewAutosaveSystem creates a new autosave system. The interval between
// saves is configured via the AUTOSAVE_TICKS environment variable. If not
// set or invalid, a default of 600 ticks is used.
func NewAutosaveSystem() *AutosaveSystem {
	interval := 600
	if v, ok := os.LookupEnv("AUTOSAVE_TICKS"); ok {
		if i, err := strconv.Atoi(v); err == nil && i > 0 {
			interval = i
		}
	}
	return &AutosaveSystem{interval: interval}
}

// Update increments the tick counter and, when the configured interval is
// reached, saves all active players via the Character model.
func (as *AutosaveSystem) Update(w *ecs.World, deltaTime float64) {
	as.counter++
	if as.counter < as.interval {
		return
	}
	as.counter = 0

	players, err := w.FindEntitiesByComponentPredicate("Player", func(i interface{}) bool { return true })
	if err != nil {
		log.Error().Err(err).Msg("autosave: failed to find players")
		return
	}

	for _, e := range players {
		player, err := ecs.GetTypedComponent[*components.Player](w, e.ID, "Player")
		if err != nil {
			log.Error().Err(err).Msg("autosave: failed to get player component")
			continue
		}

		c := models.Character{Name: player.Name}
		if player.Room != nil {
			c.X = player.Room.X
			c.Y = player.Room.Y
			c.Z = player.Room.Z
		}
		if err := c.Save(); err != nil {
			log.Error().Err(err).Msg("autosave: failed to save character")
		}
	}
}
