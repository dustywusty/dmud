package systems

import (
	"testing"
	"time"

	"dmud/internal/components"
	"dmud/internal/ecs"
)

// TestSpawnSystem_ItemSpawnRefill exercises the new SpawnTypeItem path: the sill
// fills to MaxCount, holds steady within the respawn window even after items are
// taken, and refills once the respawn timer elapses.
func TestSpawnSystem_ItemSpawnRefill(t *testing.T) {
	w := ecs.NewWorld() // no chdir: areas.json simply loads empty, which is fine here

	areaEntity := ecs.NewEntity("testarea")
	w.AddEntity(areaEntity)
	area := &components.Area{Region: "Test", Description: "A test sill."}
	w.AddComponent(&areaEntity, area)

	spawn := components.NewSpawn("testarea")
	spawn.Configs = []components.SpawnConfig{{
		Type:        components.SpawnTypeItem,
		TemplateID:  "cookie",
		MinCount:    5,
		MaxCount:    5,
		RespawnTime: time.Hour,
		Chance:      1.0,
	}}
	w.AddComponent(&areaEntity, spawn)

	ss := NewSpawnSystem()

	// First pass: the sill fills to MaxCount.
	ss.processSpawn(w, areaEntity)
	if got := countAreaItems(area, "cookie"); got != 5 {
		t.Fatalf("expected 5 cookies after first spawn, got %d", got)
	}

	// A player takes two; within the respawn window nothing replenishes.
	area.RemoveItem("cookie", 2)
	ss.processSpawn(w, areaEntity)
	if got := countAreaItems(area, "cookie"); got != 3 {
		t.Fatalf("expected no refill within the respawn window, got %d", got)
	}

	// Once the respawn timer elapses, the sill tops back up to MaxCount.
	spawn.LastItemSpawn["cookie"] = time.Now().Add(-2 * time.Hour)
	ss.processSpawn(w, areaEntity)
	if got := countAreaItems(area, "cookie"); got != 5 {
		t.Fatalf("expected refill to 5 after the respawn timer, got %d", got)
	}
}
