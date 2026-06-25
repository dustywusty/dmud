package game

import (
	"strings"
	"testing"

	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/util"
)

// merchantQuestRig builds a minimal Game with a player standing next to the
// traveling merchant (the goblin-ears quest giver), with empty inventory/quests,
// so the dialogue and give turn-in can be driven deterministically.
type merchantQuestRig struct {
	g      *Game
	player *components.Player
	client *fakeClient
	inv    *components.Inventory
	quests *components.PlayerQuests
}

func newMerchantQuestRig(t *testing.T) *merchantQuestRig {
	t.Helper()
	chdirToRepoRoot(t)
	components.InitializeQuests() // wire the goblin-ears dialogue onto the quest

	world := ecs.NewWorld()
	areaComp, err := world.GetComponent("201", "Area")
	if err != nil {
		t.Fatalf("area 201 not loaded: %v", err)
	}
	area := areaComp.(*components.Area)

	g := &Game{world: world, players: make(map[string]*ecs.Entity)}

	merchant := ecs.NewEntity()
	world.AddEntity(merchant)
	world.AddComponent(&merchant, &components.NPC{
		Name: "a traveling merchant", TemplateID: "merchant", Area: area,
	})

	client := newFakeClient("test:0")
	player := &components.Player{
		Client:         client,
		Name:           "Tester",
		Area:           area,
		CommandHistory: components.NewCommandHistory(),
		AutoComplete:   util.NewAutoComplete(),
		EnteredWorld:   true,
	}
	inv := components.NewInventory(0)
	quests := components.NewPlayerQuests()

	pe := ecs.NewEntity()
	world.AddEntity(pe)
	world.AddComponent(&pe, player)
	world.AddComponent(&pe, inv)
	world.AddComponent(&pe, quests)
	g.players[player.Name] = &pe

	return &merchantQuestRig{g: g, player: player, client: client, inv: inv, quests: quests}
}

func (r *merchantQuestRig) status() components.QuestStatus {
	return r.quests.GetQuestStatus("goblin_ears")
}

// TestGoblinQuestConfigured guards the quest content: dialogue wired, NPC set,
// and every required/reward item has a real template.
func TestGoblinQuestConfigured(t *testing.T) {
	components.InitializeQuests()
	q := components.QuestRegistry["goblin_ears"]
	if q == nil || q.Dialogue == nil {
		t.Fatal("goblin_ears quest or its dialogue is missing")
	}
	if q.NPCID != "merchant" {
		t.Errorf("quest NPCID = %q, want \"merchant\"", q.NPCID)
	}
	for _, req := range q.Requirements {
		if _, ok := components.ItemTemplates[req.ItemID]; !ok {
			t.Errorf("requirement item %q has no ItemTemplate", req.ItemID)
		}
	}
	for _, rw := range q.Rewards {
		if _, ok := components.ItemTemplates[rw.ItemID]; !ok {
			t.Errorf("reward item %q has no ItemTemplate", rw.ItemID)
		}
	}
}

// TestGoblinQuest_DialoguePath proves the intended say-keyword turn-in works
// end to end: accept, refuse when short, then complete with rewards.
func TestGoblinQuest_DialoguePath(t *testing.T) {
	r := newMerchantQuestRig(t)

	r.g.handleSayToNPC(r.player, "deal") // accept
	if r.status() != components.QuestStatusInProgress {
		t.Fatalf("after 'deal' the quest should be in progress; got %v", r.status())
	}
	r.client.drain()

	// Short of the requirement: turn-in refused, quest stays in progress.
	r.inv.AddItem(components.CreateItem("goblin_ear", 4))
	r.g.handleSayToNPC(r.player, "reward")
	if r.status() == components.QuestStatusCompleted {
		t.Fatal("quest completed with only 4/10 ears")
	}
	if out := r.client.drain(); !strings.Contains(out, "don't have everything") {
		t.Errorf("expected a 'not enough yet' reply; got %q", out)
	}

	// Enough ears: completes, consumes them, grants rewards.
	r.inv.AddItem(components.CreateItem("goblin_ear", 6)) // now 10
	r.g.handleSayToNPC(r.player, "reward")
	if r.status() != components.QuestStatusCompleted {
		t.Fatal("quest should complete once 10 ears are turned in")
	}
	if r.inv.FindItem("goblin_ear") != nil {
		t.Error("goblin ears should be consumed on completion")
	}
	if r.inv.FindItem("leather_helmet") == nil {
		t.Error("reward leather_helmet should be in the inventory")
	}
	if out := r.client.drain(); !strings.Contains(out, "Excellent work") {
		t.Errorf("expected completion message; got %q", out)
	}
}

// TestGoblinQuest_GiveTurnIn proves the fix: `give goblin ear to merchant` (the
// intuitive verb) now completes the quest instead of dead-ending.
func TestGoblinQuest_GiveTurnIn(t *testing.T) {
	r := newMerchantQuestRig(t)
	r.g.handleSayToNPC(r.player, "deal") // accept
	r.inv.AddItem(components.CreateItem("goblin_ear", 10))
	r.client.drain()

	r.g.handleGive(r.player, []string{"goblin", "ear", "to", "merchant"}, r.g)

	if r.status() != components.QuestStatusCompleted {
		t.Fatalf("give-to-merchant should complete the quest; status=%v", r.status())
	}
	if r.inv.FindItem("goblin_ear") != nil {
		t.Error("ears should be consumed by the give turn-in")
	}
	if out := r.client.drain(); strings.Contains(out, "doesn't seem interested") {
		t.Errorf("give should no longer dead-end on a quest item; got %q", out)
	}
}

// TestGoblinQuest_GiveBeforeAccept gives a useful nudge instead of a dead end
// when the player tries to hand in before accepting the quest.
func TestGoblinQuest_GiveBeforeAccept(t *testing.T) {
	r := newMerchantQuestRig(t)
	r.inv.AddItem(components.CreateItem("goblin_ear", 10))
	r.client.drain()

	r.g.handleGive(r.player, []string{"goblin", "ear", "to", "merchant"}, r.g)

	if r.status() == components.QuestStatusCompleted {
		t.Fatal("quest must not complete before it's accepted")
	}
	if out := r.client.drain(); !strings.Contains(out, "[work]") {
		t.Errorf("giving before accepting should nudge toward asking about [work]; got %q", out)
	}
	if r.inv.FindItem("goblin_ear") == nil {
		t.Error("ears must NOT be consumed when the quest isn't accepted")
	}
}
