package game

import (
	"strings"
	"testing"

	"dmud/internal/components"
)

func TestMount_RequiresOwningAHorse(t *testing.T) {
	r := newWylieTestRig(t)
	r.client.drain()

	r.g.handleMount(r.player, nil, r.g)
	out := r.client.drain()

	if !strings.Contains(out, "don't own a horse") {
		t.Fatalf("expected to be told you own no horse\n--- output ---\n%s", out)
	}
}

func TestMount_GallopRequiresMount(t *testing.T) {
	r := newWylieTestRig(t)
	r.client.drain()

	r.g.handleRide(r.player, []string{"east"}, r.g)
	out := r.client.drain()

	if !strings.Contains(out, "on foot") {
		t.Fatalf("expected to be told you're on foot\n--- output ---\n%s", out)
	}
}

func TestMount_GallopCoversMultipleRoomsAndSpendsEndurance(t *testing.T) {
	r := newWylieTestRig(t)
	r.inv.AddItem(components.CreateItem("pony", 1))

	// Mount up.
	r.client.drain()
	r.g.handleMount(r.player, []string{"pony"}, r.g)
	if !strings.Contains(r.client.drain(), "saddle of your Dapple Pony") {
		t.Fatal("expected to mount the pony")
	}

	entityID, err := r.g.getPlayerEntity(r.player)
	if err != nil {
		t.Fatalf("player entity not found: %v", err)
	}
	mountComp, err := r.g.world.GetComponent(entityID, "Mount")
	if err != nil {
		t.Fatalf("mount component missing after mounting")
	}
	mount := mountComp.(*components.Mount)
	if mount.Speed != 2 || mount.Endurance != 6 {
		t.Fatalf("unexpected pony stats: speed=%d endurance=%d", mount.Speed, mount.Endurance)
	}

	// The paddock (201) connects east to the bakery (200), which connects east to
	// the Mistwood Path (202). A speed-2 gallop east covers both rooms, landing
	// in 202.
	mistwoodComp, err := r.g.world.GetComponent("202", "Area")
	if err != nil {
		t.Fatalf("Mistwood Path (202) not loaded: %v", err)
	}
	mistwood := mistwoodComp.(*components.Area)

	r.client.drain()
	r.g.handleRide(r.player, []string{"east"}, r.g)
	out := r.client.drain()

	if !strings.Contains(out, "You ride east on your Dapple Pony") {
		t.Fatalf("expected a gallop message\n--- output ---\n%s", out)
	}
	if r.player.Area != mistwood {
		t.Fatalf("expected to gallop two rooms into the Mistwood Path (202)")
	}
	if mount.Endurance != 5 {
		t.Fatalf("expected one endurance spent on the second room, have %d", mount.Endurance)
	}

	// Dismount.
	r.client.drain()
	r.g.handleDismount(r.player, nil, r.g)
	if _, err := r.g.world.GetComponent(entityID, "Mount"); err == nil {
		t.Fatalf("expected to be dismounted")
	}
}
