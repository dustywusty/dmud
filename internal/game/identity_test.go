package game

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/persistence"
	"dmud/internal/util"
)

func (f *fakeClient) containsMessage(sub string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, m := range f.messages {
		if strings.Contains(m, sub) {
			return true
		}
	}
	return false
}

// nonTagClient is a legacy / plain-text client (SupportsTags() == false), used to
// confirm IDENTITY| protocol frames are hidden from clients that can't use them.
type nonTagClient struct {
	messages []string
}

func (c *nonTagClient) CloseConnection() error { return nil }
func (c *nonTagClient) HandleRequest()         {}
func (c *nonTagClient) SendMessage(msg string) { c.messages = append(c.messages, msg) }
func (c *nonTagClient) RemoteAddr() string     { return "legacy:0" }
func (c *nonTagClient) SupportsPrompt() bool   { return false }
func (c *nonTagClient) SupportsTags() bool     { return false }

func newMemoryGame(t *testing.T) *Game {
	t.Helper()
	chdirToRepoRoot(t)
	return &Game{
		world:       ecs.NewWorld(),
		players:     make(map[string]*ecs.Entity),
		store:       persistence.NewMemoryStore(),
		sessionKeys: make(map[common.Client]string),
	}
}

func registerPlayer(t *testing.T, g *Game, client common.Client, name, areaID string) (*components.Player, common.EntityID) {
	t.Helper()
	var area *components.Area
	if areaID != "" {
		comp, err := g.world.GetComponent(common.EntityID(areaID), "Area")
		if err != nil {
			t.Fatalf("area %s not loaded: %v", areaID, err)
		}
		area = comp.(*components.Area)
	}
	player := &components.Player{
		Client:         client,
		Name:           name,
		Area:           area,
		CommandHistory: components.NewCommandHistory(),
		AutoComplete:   util.NewAutoComplete(),
		EnteredWorld:   true,
	}
	entity := ecs.NewEntity()
	g.world.AddEntity(entity)
	g.world.AddComponent(&entity, player)
	g.world.AddComponent(&entity, components.NewInventory(0))
	g.world.AddComponent(&entity, components.NewFactions())
	g.world.AddComponent(&entity, components.NewHealth(1))
	g.world.AddComponent(&entity, components.NewExperience())
	g.players[name] = &entity
	return player, entity.ID
}

func TestIdentity_HiddenFromLegacyClients(t *testing.T) {
	legacy := &nonTagClient{}
	p := &components.Player{Client: legacy, Name: "Legacy"}

	p.Broadcast(util.IdentityMessage("abc-123"))
	if len(legacy.messages) != 0 {
		t.Fatalf("legacy client should not receive IDENTITY frames, got %v", legacy.messages)
	}

	p.Broadcast("hello there")
	if len(legacy.messages) != 1 || legacy.messages[0] != "hello there" {
		t.Fatalf("legacy client should still receive plain messages, got %v", legacy.messages)
	}
}

func TestEnsureIdentity_AutoAssignsAndPersists(t *testing.T) {
	g := newMemoryGame(t)
	client := newFakeClient("web:0")
	player, _ := registerPlayer(t, g, client, "Auto", "300")
	client.drain()

	id := g.ensureIdentity(player)
	if id == "" {
		t.Fatal("expected a persistence id to be minted")
	}
	if got := g.clientPersistenceKey(client); got != id {
		t.Fatalf("session not linked to the minted id: %q vs %q", got, id)
	}
	if !strings.Contains(client.drain(), util.IdentityMessage(id)) {
		t.Fatal("client should be told its identity so it can store it")
	}

	saved, err := g.store.LoadPlayer(context.Background(), id)
	if err != nil || saved == nil {
		t.Fatalf("character should have been auto-saved: %v", err)
	}
	if saved.Name != "Auto" || saved.AreaID != "300" {
		t.Fatalf("unexpected saved state: name=%q area=%q", saved.Name, saved.AreaID)
	}

	// Idempotent: a second call keeps the same id (no new character).
	if again := g.ensureIdentity(player); again != id {
		t.Fatalf("identity should be stable across calls: %q vs %q", again, id)
	}
}

func TestLogin_RestoresSavedCharacter(t *testing.T) {
	g := newMemoryGame(t)

	// A character the player saved in a previous session.
	if err := g.store.SavePlayer(context.Background(), "id-123", &persistence.PlayerState{
		Version:  playerStateVersion,
		Name:     "Restored",
		AreaID:   "300",
		Factions: map[string]int{"cinderhollow": 55},
	}); err != nil {
		t.Fatalf("seed saved character: %v", err)
	}

	client := newFakeClient("web:1")
	player, entityID := registerPlayer(t, g, client, "Wanderer", "200")
	client.drain()

	// The client replays its stored id on reconnect.
	g.HandleLogin(player, "id-123")

	if player.Name != "Restored" {
		t.Fatalf("expected name to be restored, got %q", player.Name)
	}
	area300Comp, _ := g.world.GetComponent(common.EntityID("300"), "Area")
	if player.Area != area300Comp.(*components.Area) {
		t.Fatal("expected to be moved to the saved area (300)")
	}
	factionsComp, err := g.world.GetComponent(entityID, "Factions")
	if err != nil {
		t.Fatalf("factions component missing: %v", err)
	}
	if got := factionsComp.(*components.Factions).Get("cinderhollow"); got != 55 {
		t.Fatalf("expected restored cinderhollow reputation 55, got %d", got)
	}
	if g.clientPersistenceKey(client) != "id-123" {
		t.Fatal("session should be linked to the restored id so it keeps autosaving")
	}
	if !strings.Contains(client.drain(), util.IdentityMessage("id-123")) {
		t.Fatal("client should be told its identity after a successful login")
	}
}

// TestEnterWorld_AutoPersistsToFile drives the real game loop with the file store
// to prove a fresh web client is auto-assigned an identity and written to disk,
// with no manual `save` -- the end-to-end path enterWorld -> ensureIdentity ->
// FileStore.
func TestEnterWorld_AutoPersistsToFile(t *testing.T) {
	chdirToRepoRoot(t)
	dataDir := t.TempDir()
	t.Setenv("DMUD_PERSISTENCE", "file")
	t.Setenv("DMUD_DATA_DIR", dataDir)

	if err := components.LoadNPCTemplates("./resources/npcs.json"); err != nil {
		t.Fatalf("load npc templates: %v", err)
	}

	g := NewGame()
	c := newFakeClient("web:integration")
	g.AddPlayerChan <- c

	// Wait out the login grace + enterWorld, then for the auto-save to land.
	var savedFiles []string
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		savedFiles, _ = filepath.Glob(filepath.Join(dataDir, "players", "*.json"))
		if c.containsMessage("IDENTITY|") && len(savedFiles) > 0 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !c.containsMessage("IDENTITY|") {
		t.Fatal("a fresh web client should receive an IDENTITY frame")
	}
	if len(savedFiles) == 0 {
		t.Fatalf("a fresh web client should be auto-saved under %s/players/", dataDir)
	}
}
