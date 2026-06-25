package game

import (
	"strings"
	"testing"

	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/util"
)

// drain returns all messages sent to the client so far and clears the buffer,
// so each step of a test can assert on just that step's output.
func (f *fakeClient) drain() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := strings.Join(f.messages, "\n")
	f.messages = nil
	return out
}

// wylieTestRig builds a minimal Game (no running loop) with a player standing in
// Wylie's riverside paddock (area 201, loaded from resources/areas.json) next to
// the Wylie NPC, so the give/buy/list handlers can be driven deterministically.
type wylieTestRig struct {
	g        *Game
	player   *components.Player
	client   *fakeClient
	inv      *components.Inventory
	factions *components.Factions
}

func newWylieTestRig(t *testing.T) *wylieTestRig {
	t.Helper()
	chdirToRepoRoot(t)

	world := ecs.NewWorld() // loads resources/areas.json, including the paddock
	areaComp, err := world.GetComponent("201", "Area")
	if err != nil {
		t.Fatalf("paddock area 201 not loaded: %v", err)
	}
	paddock := areaComp.(*components.Area)

	g := &Game{
		world:   world,
		players: make(map[string]*ecs.Entity),
	}

	// Place Wylie in her paddock.
	wylieEntity := ecs.NewEntity()
	world.AddEntity(wylieEntity)
	world.AddComponent(&wylieEntity, &components.NPC{
		Name:       "Wylie",
		TemplateID: "wylie",
		Area:       paddock,
		Behavior:   components.BehaviorMerchant,
	})

	// Create the player standing in the paddock.
	client := newFakeClient("test:0")
	player := &components.Player{
		Client:         client,
		Name:           "Tester",
		Area:           paddock,
		CommandHistory: components.NewCommandHistory(),
		AutoComplete:   util.NewAutoComplete(),
		EnteredWorld:   true,
	}
	inv := components.NewInventory(0)
	factions := components.NewFactions()

	playerEntity := ecs.NewEntity()
	world.AddEntity(playerEntity)
	world.AddComponent(&playerEntity, player)
	world.AddComponent(&playerEntity, inv)
	world.AddComponent(&playerEntity, factions)

	g.players[player.Name] = &playerEntity

	return &wylieTestRig{g: g, player: player, client: client, inv: inv, factions: factions}
}

func TestWylie_GiveCookiesRaisesFaction(t *testing.T) {
	r := newWylieTestRig(t)

	r.inv.AddItem(components.CreateItem("cookie", 5))
	r.client.drain()

	r.g.handleGive(r.player, []string{"cookies", "to", "wylie"}, r.g)
	out := r.client.drain()

	if got := r.factions.Get("cinderhollow"); got != 25 {
		t.Fatalf("expected 25 reputation after giving 5 cookies (5 each), got %d", got)
	}
	if r.inv.FindItem("cookie") != nil {
		t.Fatalf("cookies should have been handed over, but some remain in inventory")
	}
	for _, want := range []string{"You give Cookies x5 to Wylie", "rises by 25", "Acquaintance"} {
		if !strings.Contains(out, want) {
			t.Fatalf("give output missing %q\n--- output ---\n%s", want, out)
		}
	}
}

func TestWylie_RejectsNonCookies(t *testing.T) {
	r := newWylieTestRig(t)

	r.inv.AddItem(components.CreateItem("rat_tail", 1))
	r.client.drain()

	r.g.handleGive(r.player, []string{"rat", "tail", "to", "wylie"}, r.g)
	out := r.client.drain()

	if r.factions.Get("cinderhollow") != 0 {
		t.Fatalf("giving a non-cookie should not change reputation")
	}
	if r.inv.FindItem("rat_tail") == nil {
		t.Fatalf("a rejected gift should stay in the player's inventory")
	}
	if !strings.Contains(out, "only trades in cookies") {
		t.Fatalf("expected Wylie to decline non-cookies\n--- output ---\n%s", out)
	}
}

func TestWylie_BuyGatedByFaction(t *testing.T) {
	r := newWylieTestRig(t)

	// Plenty of gold, but no standing yet.
	r.inv.AddItem(components.CreateItem("gold_coin", 300))
	r.client.drain()

	r.g.handleBuy(r.player, []string{"pony"}, r.g)
	out := r.client.drain()

	if !strings.Contains(out, "Friend") {
		t.Fatalf("buying below the faction gate should mention the required rank\n--- output ---\n%s", out)
	}
	if r.inv.FindItem("pony") != nil {
		t.Fatalf("no horse should be sold below the faction gate")
	}
	if got := countInventoryItem(r.inv, "gold_coin"); got != 300 {
		t.Fatalf("gold should not be spent on a refused purchase, have %d", got)
	}
}

func TestWylie_BuyOnceTrusted(t *testing.T) {
	r := newWylieTestRig(t)

	// Earn the right to shop, and bring coin.
	r.factions.Set("cinderhollow", 40) // "Friend"
	r.inv.AddItem(components.CreateItem("gold_coin", 100))
	r.client.drain()

	r.g.handleBuy(r.player, []string{"pony"}, r.g)
	out := r.client.drain()

	if !strings.Contains(out, "You buy Dapple Pony from Wylie for 80 gold") {
		t.Fatalf("expected a successful purchase\n--- output ---\n%s", out)
	}
	if r.inv.FindItem("pony") == nil {
		t.Fatalf("purchased pony should be in the inventory")
	}
	if got := countInventoryItem(r.inv, "gold_coin"); got != 20 {
		t.Fatalf("expected 20 gold left after an 80-gold purchase, got %d", got)
	}

	// The courser needs Honored standing, which a Friend hasn't reached.
	r.client.drain()
	r.g.handleBuy(r.player, []string{"courser"}, r.g)
	out = r.client.drain()
	if !strings.Contains(out, "Honored") {
		t.Fatalf("expected the courser to require Honored standing\n--- output ---\n%s", out)
	}

	// And too poor for a second pony (80 > 20 remaining).
	r.client.drain()
	r.g.handleBuy(r.player, []string{"pony"}, r.g)
	out = r.client.drain()
	if !strings.Contains(out, "you've only got 20") {
		t.Fatalf("expected an affordability refusal\n--- output ---\n%s", out)
	}
}

func TestWylie_ListShowsWaresAndStanding(t *testing.T) {
	r := newWylieTestRig(t)
	r.client.drain()

	r.g.handleList(r.player, nil, r.g)
	out := r.client.drain()

	for _, want := range []string{"Wylie's Riverside Paddock", "Dapple Pony", "80 gold", "Midnight Courser", "Stranger"} {
		if !strings.Contains(out, want) {
			t.Fatalf("list output missing %q\n--- output ---\n%s", want, out)
		}
	}
}
