package game

import (
	"strings"
	"testing"
)

func TestClassesCommand(t *testing.T) {
	g, player, _, client := newClericRig(t, 10)
	client.drain()

	g.handleClasses(player, nil, g)
	out := client.drain()

	for _, want := range []string{"Pyromancer", "Cleric", "Warlock", "Druid", "pyromancy", "restoration", "spark", "mend"} {
		if !strings.Contains(out, want) {
			t.Errorf("classes output missing %q\n--- output ---\n%s", want, out)
		}
	}
}

func TestLunarSchool(t *testing.T) {
	if moon := spellsInSchool("lunar"); len(moon) == 0 {
		t.Fatal("the lunar school should have spells")
	}
	mf := spellRegistry["moonfire"]
	if mf == nil || mf.School != "lunar" || mf.Cost != 10 {
		t.Errorf("moonfire mis-registered: %+v", mf)
	}
	// The Druid draws on restoration + lunar.
	druid, ok := classByKey("druid")
	if !ok || strings.Join(druid.Schools, ",") != "restoration,lunar" {
		t.Errorf("druid schools = %v, want [restoration lunar]", druid.Schools)
	}
}

func TestSpellsInSchool(t *testing.T) {
	fire := spellsInSchool("pyromancy")
	if len(fire) == 0 {
		t.Fatal("pyromancy school should have spells")
	}
	// Ordered by level: spark (1) before immolate (15).
	if fire[0].Name != "spark" || fire[len(fire)-1].Name != "immolate" {
		t.Errorf("pyromancy not ordered by level: first=%s last=%s", fire[0].Name, fire[len(fire)-1].Name)
	}
}
