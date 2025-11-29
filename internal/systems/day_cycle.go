package systems

import (
	"dmud/internal/components"
	"dmud/internal/ecs"
	"time"

	"github.com/rs/zerolog/log"
)

type DayCycleSystem struct {
	lastUpdate time.Time
	dayCycle   *components.DayCycle
	broadcast  func(string)
}

func NewDayCycleSystem(broadcast func(string)) *DayCycleSystem {
	return &DayCycleSystem{
		lastUpdate: time.Now(),
		dayCycle:   components.NewDayCycle(),
		broadcast:  broadcast,
	}
}

func (dcs *DayCycleSystem) GetDayCycle() *components.DayCycle {
	return dcs.dayCycle
}

func (dcs *DayCycleSystem) Update(w *ecs.World, deltaTime float64) {
	now := time.Now()
	elapsed := now.Sub(dcs.lastUpdate)
	dcs.lastUpdate = now

	dcs.dayCycle.Lock()
	defer dcs.dayCycle.Unlock()

	dcs.dayCycle.ElapsedTime += elapsed

	// Check if we need to transition to next period
	periodDuration := dcs.dayCycle.GetCurrentPeriodDuration()
	if dcs.dayCycle.ElapsedTime >= periodDuration {
		dcs.dayCycle.ElapsedTime -= periodDuration
		oldTime := dcs.dayCycle.CurrentTime
		dcs.advanceTime()
		dcs.announceTransition(oldTime, dcs.dayCycle.CurrentTime)
	}
}

func (dcs *DayCycleSystem) advanceTime() {
	switch dcs.dayCycle.CurrentTime {
	case components.Dawn:
		dcs.dayCycle.CurrentTime = components.Day
	case components.Day:
		dcs.dayCycle.CurrentTime = components.Dusk
	case components.Dusk:
		dcs.dayCycle.CurrentTime = components.Night
	case components.Night:
		dcs.dayCycle.CurrentTime = components.Dawn
		dcs.dayCycle.DayNumber++
		dcs.dayCycle.CycleStart = time.Now()
	}
}

func (dcs *DayCycleSystem) announceTransition(from, to components.TimeOfDay) {
	var message string
	switch to {
	case components.Dawn:
		message = "\n[The first rays of sunlight peek over the horizon. A new day begins.]\n"
		log.Info().Msgf("Day %d has begun", dcs.dayCycle.DayNumber)
	case components.Day:
		message = "\n[The sun rises fully into the sky. It is now daytime.]\n"
	case components.Dusk:
		message = "\n[The sun begins to set, casting long shadows across the land.]\n"
	case components.Night:
		message = "\n[Darkness falls as night takes hold. The stars emerge overhead.]\n"
	}

	if dcs.broadcast != nil && message != "" {
		dcs.broadcast(message)
	}
}
