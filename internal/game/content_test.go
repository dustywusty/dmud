package game

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
)

// TestBakeryContentLoads validates that the new resource files parse and wire up:
// Wylie's NPC template, the bakery/stable area and its exit, and the spawn block
// that produces both Wylie and the windowsill cookies.
func TestBakeryContentLoads(t *testing.T) {
	chdirToRepoRoot(t)

	if err := components.LoadNPCTemplates("./resources/npcs.json"); err != nil {
		t.Fatalf("load npc templates: %v", err)
	}
	wylie, ok := components.NPCTemplates["wylie"]
	if !ok {
		t.Fatal("wylie NPC template not loaded")
	}
	if wylie.Name != "Wylie" || !wylie.Stationary {
		t.Fatalf("unexpected wylie template: name=%q stationary=%v", wylie.Name, wylie.Stationary)
	}

	// spawns.json: the bakery (200) spawns cookies; the paddock (201) spawns Wylie.
	data, err := os.ReadFile("./resources/spawns.json")
	if err != nil {
		t.Fatalf("read spawns.json: %v", err)
	}
	var areaSpawns []areaSpawnJSON
	if err := json.Unmarshal(data, &areaSpawns); err != nil {
		t.Fatalf("parse spawns.json: %v", err)
	}
	spawnsFor := func(areaID string) []spawnConfigJSON {
		for i := range areaSpawns {
			if areaSpawns[i].AreaID == areaID {
				return areaSpawns[i].Spawns
			}
		}
		return nil
	}
	hasSpawn := func(spawns []spawnConfigJSON, templateID, spawnType string) bool {
		for _, s := range spawns {
			if s.TemplateID == templateID && (spawnType == "" || strings.EqualFold(s.Type, spawnType)) {
				return true
			}
		}
		return false
	}

	bakerySpawns := spawnsFor("200")
	if bakerySpawns == nil {
		t.Fatal("spawns.json has no block for the bakery (area 200)")
	}
	if !hasSpawn(bakerySpawns, "cookie", "item") {
		t.Error("the bakery (200) should spawn cookie items (type: item)")
	}
	if hasSpawn(bakerySpawns, "wylie", "") {
		t.Error("Wylie should live in the wild (201), not the bakery (200)")
	}

	paddockSpawns := spawnsFor("201")
	if paddockSpawns == nil {
		t.Fatal("spawns.json has no block for the paddock (area 201)")
	}
	if !hasSpawn(paddockSpawns, "wylie", "") {
		t.Error("the paddock (201) should spawn Wylie")
	}
	if hasSpawn(paddockSpawns, "cookie", "item") {
		t.Error("cookies belong in the bakery (200), not the paddock (201)")
	}

	// areas.json (loaded by NewWorld): the bakery and paddock exist and link up.
	w := ecs.NewWorld()
	hasExit := func(areaID, direction, dest string) bool {
		comp, err := w.GetComponent(common.EntityID(areaID), "Area")
		if err != nil {
			return false
		}
		for _, exit := range comp.(*components.Area).Exits {
			if exit.Direction == direction && exit.AreaID == dest {
				return true
			}
		}
		return false
	}
	descOf := func(areaID string) string {
		comp, err := w.GetComponent(common.EntityID(areaID), "Area")
		if err != nil {
			t.Fatalf("area %s not loaded: %v", areaID, err)
		}
		return comp.(*components.Area).Description
	}

	if !strings.Contains(descOf("200"), "BAKEHOUSE") {
		t.Errorf("area 200 should be the bakehouse: %q", descOf("200"))
	}
	if !strings.Contains(descOf("201"), "PADDOCK") {
		t.Errorf("area 201 should be the paddock: %q", descOf("201"))
	}
	if !hasExit("2", "west", "202") {
		t.Error("area 2 should lead west onto the Mistwood Path (202)")
	}
	if !hasExit("202", "west", "200") {
		t.Error("the Mistwood Path (202) should lead west into the bakery (200)")
	}
	if !hasExit("200", "west", "201") {
		t.Error("the bakery (200) should lead west into the paddock (201)")
	}
}

// TestWorldGraph_NoDanglingExits guards the expanded map: every exit must point
// at a real area, and the town hub must connect to the wider world.
func TestWorldGraph_NoDanglingExits(t *testing.T) {
	chdirToRepoRoot(t)

	data, err := os.ReadFile("./resources/areas.json")
	if err != nil {
		t.Fatalf("read areas.json: %v", err)
	}
	var defs []struct {
		ID    string            `json:"id"`
		Exits map[string]string `json:"exits"`
	}
	if err := json.Unmarshal(data, &defs); err != nil {
		t.Fatalf("parse areas.json: %v", err)
	}

	exits := make(map[string]map[string]string, len(defs))
	for _, d := range defs {
		exits[d.ID] = d.Exits
	}

	for id, areaExits := range exits {
		for dir, dest := range areaExits {
			if _, ok := exits[dest]; !ok {
				t.Errorf("area %s has a %s exit to nonexistent area %q", id, dir, dest)
			}
		}
	}

	// BFS from the town square (300). Area 99 is intentionally hidden, so we only
	// assert the landmarks the town should connect to are reachable.
	reach := map[string]bool{"300": true}
	queue := []string{"300"}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, dest := range exits[cur] {
			if !reach[dest] {
				reach[dest] = true
				queue = append(queue, dest)
			}
		}
	}
	for _, want := range []string{"1", "7", "100", "200", "201", "311", "313"} {
		if !reach[want] {
			t.Errorf("area %s is not reachable from the town square (300)", want)
		}
	}
}

// TestSpawns_RatsInSewerGuardsInTown verifies the spawn relocation: rats now
// live under the town, and the guard and merchant moved into Ravenmoor.
func TestSpawns_RatsInSewerGuardsInTown(t *testing.T) {
	chdirToRepoRoot(t)

	data, err := os.ReadFile("./resources/spawns.json")
	if err != nil {
		t.Fatalf("read spawns.json: %v", err)
	}
	var areaSpawns []areaSpawnJSON
	if err := json.Unmarshal(data, &areaSpawns); err != nil {
		t.Fatalf("parse spawns.json: %v", err)
	}

	byArea := make(map[string][]spawnConfigJSON, len(areaSpawns))
	for _, a := range areaSpawns {
		byArea[a.AreaID] = a.Spawns
	}
	has := func(areaID, templateID string) bool {
		for _, s := range byArea[areaID] {
			if s.TemplateID == templateID {
				return true
			}
		}
		return false
	}

	for _, sewer := range []string{"311", "312", "313"} {
		if !has(sewer, "rat") {
			t.Errorf("sewer area %s should spawn rats", sewer)
		}
	}
	if has("1", "rat") || has("2", "rat") {
		t.Error("rats should no longer spawn in the overworld (areas 1/2) -- they moved to the sewer")
	}
	if !has("300", "guard") {
		t.Error("the town square (300) should be watched by a guard")
	}
	if !has("301", "merchant") {
		t.Error("the merchant should set up shop on Market Row (301)")
	}
}
