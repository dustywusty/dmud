package components

import (
	"math"
	"sync"
)

type Experience struct {
	sync.RWMutex
	Current int
	Level   int
}

func NewExperience() *Experience {
	return &Experience{
		Current: 0,
		Level:   1,
	}
}

func (e *Experience) Type() string {
	return "Experience"
}

func (e *Experience) GetLevel() int {
	e.RLock()
	defer e.RUnlock()
	return e.Level
}

func (e *Experience) GetCurrent() int {
	e.RLock()
	defer e.RUnlock()
	return e.Current
}

func (e *Experience) GetRequiredXP() int {
	e.RLock()
	defer e.RUnlock()
	return CalculateRequiredXP(e.Level)
}

func CalculateRequiredXP(level int) int {
	return int(math.Floor(100 * math.Pow(float64(level), 1.5)))
}

func (e *Experience) AddXP(amount int) (leveledUp bool, newLevel int) {
	e.Lock()
	defer e.Unlock()

	e.Current += amount
	oldLevel := e.Level

	for e.Current >= CalculateRequiredXP(e.Level) {
		e.Current -= CalculateRequiredXP(e.Level)
		e.Level++
	}

	if e.Level > oldLevel {
		return true, e.Level
	}

	return false, e.Level
}

func GetLevelScaling(level int) float64 {
	if level <= 1 {
		return 1.0
	}
	return 1.0 + (float64(level-1) * 0.1)
}
