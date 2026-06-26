package components

import "testing"

func TestEnduranceBonusFromCON(t *testing.T) {
	st := NewStats() // human: CON at the base value, so no bonus
	if got := st.EnduranceBonus(); got != 0 {
		t.Errorf("base CON should grant 0 bonus endurance, got %d", got)
	}

	st.Set(CON, StatBase+20) // +20 points × 5 = +100
	if got := st.EnduranceBonus(); got != 100 {
		t.Errorf("CON +20 endurance bonus = %d, want 100", got)
	}

	// An ogre's racial CON (+10) alone gives a noticeably deeper pool.
	ogre := NewStatsForRace("ogre")
	if ogre.EnduranceBonus() <= 0 {
		t.Errorf("an ogre should start with bonus endurance from CON, got %d", ogre.EnduranceBonus())
	}
}

func TestEnduranceBaseScalesWithLevel(t *testing.T) {
	if EnduranceBaseForLevel(5) <= EnduranceBaseForLevel(1) {
		t.Error("endurance base pool should grow with level")
	}
}
