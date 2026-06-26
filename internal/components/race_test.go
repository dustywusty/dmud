package components

import "testing"

func TestNewStatsForRace(t *testing.T) {
	ogre := NewStatsForRace("ogre")
	if ogre.Race != "Ogre" || ogre.Size != SizeLarge {
		t.Errorf("ogre race/size = %s/%v, want Ogre/Large", ogre.Race, ogre.Size)
	}
	if ogre.Get(STR) != 20 || ogre.Get(INT) != 2 {
		t.Errorf("ogre STR/INT = %d/%d, want 20/2", ogre.Get(STR), ogre.Get(INT))
	}
	if ogre.HPBonus() != 30 { // CON 20 → (20-10)*3
		t.Errorf("ogre HP bonus = %d, want 30", ogre.HPBonus())
	}
	if ogre.ArcaneFactor() >= 1.0 { // INT 2 → weaker than base
		t.Errorf("a dim ogre should cast below baseline; arcane factor = %v", ogre.ArcaneFactor())
	}

	gob := NewStatsForRace("goblin")
	if gob.Size != SizeSmall || gob.Get(DEX) != 16 || gob.Get(STR) != 6 {
		t.Errorf("goblin size/DEX/STR = %v/%d/%d, want Small/16/6", gob.Size, gob.Get(DEX), gob.Get(STR))
	}

	// Unknown race falls back to a balanced Human.
	h := NewStatsForRace("definitely-not-a-race")
	if h.Race != "Human" || h.Size != SizeMedium || h.Get(STR) != statBase {
		t.Errorf("unknown race should fall back to Human; got %s/%v/STR%d", h.Race, h.Size, h.Get(STR))
	}
}
