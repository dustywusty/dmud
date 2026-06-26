package components

import (
	"testing"
	"time"

	"dmud/internal/common"
)

func TestStatusEffectsControlledBy(t *testing.T) {
	master := common.EntityID("master")
	other := common.EntityID("other")

	se := NewStatusEffects()
	if se.ControlledBy(master) {
		t.Error("no effects → not controlled")
	}

	se.AddEffect(StatusEffect{
		Type: StatusEffectCharmed, Name: "Charmed",
		AppliedAt: time.Now(), Duration: time.Minute, SourceEntityID: master,
	})
	if !se.ControlledBy(master) {
		t.Error("charmed by master → controlled")
	}
	if se.ControlledBy(other) {
		t.Error("must not be controlled by a different caster")
	}

	expired := NewStatusEffects()
	expired.AddEffect(StatusEffect{
		Type:      StatusEffectControlledUndead,
		AppliedAt: time.Now().Add(-time.Hour), Duration: time.Minute, SourceEntityID: master,
	})
	if expired.ControlledBy(master) {
		t.Error("an expired control effect → not controlled")
	}

	blessing := NewStatusEffects()
	blessing.AddEffect(StatusEffect{
		Type:      StatusEffectGuardBlessing,
		AppliedAt: time.Now(), Duration: time.Minute, SourceEntityID: master,
	})
	if blessing.ControlledBy(master) {
		t.Error("a non-control effect must not imply control")
	}
}
