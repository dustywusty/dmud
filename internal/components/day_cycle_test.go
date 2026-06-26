package components

import "testing"

func TestMoonPhaseCycle(t *testing.T) {
	if got := MoonPhaseForDay(1); got != NewMoon {
		t.Errorf("day 1 = %v, want new moon", got)
	}
	if got := MoonPhaseForDay(5); got != FullMoon {
		t.Errorf("day 5 = %v, want full moon", got)
	}
	if got := MoonPhaseForDay(9); got != NewMoon {
		t.Errorf("day 9 should wrap to new moon, got %v", got)
	}
	if MoonPhaseForDay(2) != MoonPhaseForDay(10) {
		t.Error("phase should repeat every 8 days")
	}
}

func TestLunarPowerFullBeatsNew(t *testing.T) {
	full := &DayCycle{DayNumber: 5, CurrentTime: Day} // full moon, daylight
	newm := &DayCycle{DayNumber: 1, CurrentTime: Day} // new moon, daylight
	if full.LunarPower() <= newm.LunarPower() {
		t.Errorf("full moon (%.2f) should beat new moon (%.2f)", full.LunarPower(), newm.LunarPower())
	}
	if full.LunarPower() < 1.4 {
		t.Errorf("full moon should be ~1.5x, got %.2f", full.LunarPower())
	}

	// Night rides the moon higher than day at the same phase.
	dayFull := &DayCycle{DayNumber: 5, CurrentTime: Day}
	nightFull := &DayCycle{DayNumber: 5, CurrentTime: Night}
	if nightFull.LunarPower() <= dayFull.LunarPower() {
		t.Error("night should boost lunar power over day at the same phase")
	}
}
