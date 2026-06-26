package game

import (
	"testing"
	"time"

	"dmud/internal/components"
)

func TestBuffDuration(t *testing.T) {
	base := 15 * time.Minute
	st := components.NewStats() // human: every stat at the base value

	if d := buffDuration(base, st, components.WIS); d != base {
		t.Errorf("base-skill duration = %v, want %v", d, base)
	}

	st.Set(components.WIS, components.StatBase+30) // +30 points × 30s = +15m
	if got, want := buffDuration(base, st, components.WIS), base+15*time.Minute; got != want {
		t.Errorf("skilled duration = %v, want %v", got, want)
	}

	st.Set(components.WIS, 100) // far above the soft target
	if d := buffDuration(base, st, components.WIS); d != 45*time.Minute {
		t.Errorf("master duration = %v, want the 45m cap", d)
	}

	if d := buffDuration(base, nil, components.WIS); d != base {
		t.Errorf("nil-stats duration = %v, want %v", d, base)
	}
}
