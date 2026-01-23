package components

import (
	"sync"
	"time"
)

type TimeOfDay int

const (
	Dawn TimeOfDay = iota
	Day
	Dusk
	Night
)

func (t TimeOfDay) String() string {
	switch t {
	case Dawn:
		return "dawn"
	case Day:
		return "day"
	case Dusk:
		return "dusk"
	case Night:
		return "night"
	default:
		return "unknown"
	}
}

// DayCycle tracks the world's day/night cycle
// A full day cycle takes 2 hours (7200 seconds):
// - Dawn:  10 minutes (600s)  - 8.3%
// - Day:   50 minutes (3000s) - 41.7%
// - Dusk:  10 minutes (600s)  - 8.3%
// - Night: 50 minutes (3000s) - 41.7%

type DayCycle struct {
	sync.RWMutex
	CurrentTime TimeOfDay
	ElapsedTime time.Duration // Time elapsed in current period
	CycleStart  time.Time     // When the current full cycle started
	DayNumber   int           // How many full days have passed
}

// Period durations
const (
	DawnDuration    = 10 * time.Minute
	DayDuration     = 50 * time.Minute
	DuskDuration    = 10 * time.Minute
	NightDuration   = 50 * time.Minute
	FullDayDuration = DawnDuration + DayDuration + DuskDuration + NightDuration // 2 hours
)

func NewDayCycle() *DayCycle {
	return &DayCycle{
		CurrentTime: Dawn,
		ElapsedTime: 0,
		CycleStart:  time.Now(),
		DayNumber:   1,
	}
}

func (dc *DayCycle) Type() string {
	return "DayCycle"
}

// GetCurrentPeriodDuration returns the duration of the current time period
func (dc *DayCycle) GetCurrentPeriodDuration() time.Duration {
	switch dc.CurrentTime {
	case Dawn:
		return DawnDuration
	case Day:
		return DayDuration
	case Dusk:
		return DuskDuration
	case Night:
		return NightDuration
	default:
		return DayDuration
	}
}

// GetTimeRemaining returns how much time is left in the current period
func (dc *DayCycle) GetTimeRemaining() time.Duration {
	dc.RLock()
	defer dc.RUnlock()
	return dc.GetCurrentPeriodDuration() - dc.ElapsedTime
}

// IsLight returns true if it's currently light outside (dawn or day)
func (dc *DayCycle) IsLight() bool {
	dc.RLock()
	defer dc.RUnlock()
	return dc.CurrentTime == Dawn || dc.CurrentTime == Day
}

// IsDark returns true if it's currently dark outside (dusk or night)
func (dc *DayCycle) IsDark() bool {
	dc.RLock()
	defer dc.RUnlock()
	return dc.CurrentTime == Dusk || dc.CurrentTime == Night
}

// GetDescription returns a descriptive string for the current time
func (dc *DayCycle) GetDescription() string {
	dc.RLock()
	defer dc.RUnlock()
	switch dc.CurrentTime {
	case Dawn:
		return "The sun is rising, casting long shadows across the land."
	case Day:
		return "The sun shines brightly overhead."
	case Dusk:
		return "The sun is setting, painting the sky in shades of orange and red."
	case Night:
		return "The moon and stars illuminate the dark sky."
	default:
		return "The sky is unclear."
	}
}
