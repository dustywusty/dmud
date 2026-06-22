package game

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/persistence"
	"dmud/internal/systems"
	"dmud/internal/util"

	"github.com/rs/zerolog/log"
)

type ClientCommand struct {
	Client common.Client
	Cmd    string
	Args   []string
}

type CommandHandler func(player *components.Player, args []string, game *Game)

type Command struct {
	Name        string
	Aliases     []string
	Handler     CommandHandler
	Description string
	Hidden      bool
}

var commandRegistry = make(map[string]*Command)

type Game struct {
	defaultArea *components.Area

	players   map[string]*ecs.Entity
	playersMu sync.RWMutex

	world *ecs.World

	dayCycleSystem *systems.DayCycleSystem

	store          persistence.Store
	persistenceErr error
	adminKeys      map[string]bool

	sessionKeys   map[common.Client]string
	sessionKeysMu sync.RWMutex

	lastAutosave     time.Time
	autosaveInterval time.Duration

	AddPlayerChan      chan common.Client
	RemovePlayerChan   chan common.Client
	ExecuteCommandChan chan ClientCommand
	enterWorldChan     chan common.Client

	// Server stats
	StartTime      time.Time
	UniqueIPs      map[string]bool
	UniqueIPsMu    sync.RWMutex
	TotalConnects  int
	TotalConnectMu sync.RWMutex
}

func NewGame() *Game {
	combatSystem := &systems.CombatSystem{}
	movementSystem := &systems.MovementSystem{}
	spawnSystem := systems.NewSpawnSystem()
	aiSystem := systems.NewAISystem()
	corpseSystem := systems.NewCorpseSystem()
	statusEffectSystem := systems.NewStatusEffectSystem()

	world := ecs.NewWorld()
	world.AddSystem(combatSystem)
	world.AddSystem(movementSystem)
	world.AddSystem(spawnSystem)
	world.AddSystem(aiSystem)
	world.AddSystem(corpseSystem)
	world.AddSystem(statusEffectSystem)

	defaultAreaUntyped, err := world.GetComponent("1", "Area")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get default area")
	}

	defaultArea, ok := defaultAreaUntyped.(*components.Area)
	if !ok {
		log.Fatal().Msg("Failed to cast default area to *components.Area")
	}

	store, err := persistence.NewStoreFromEnv()
	var persistenceErr error
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize persistence store")
		persistenceErr = err
		store = &persistence.NoopStore{}
	}

	game := &Game{
		defaultArea:        defaultArea,
		players:            make(map[string]*ecs.Entity),
		world:              world,
		AddPlayerChan:      make(chan common.Client, 64),
		RemovePlayerChan:   make(chan common.Client, 64),
		ExecuteCommandChan: make(chan ClientCommand, 256),
		enterWorldChan:     make(chan common.Client, 64),
		StartTime:          time.Now(),
		UniqueIPs:          make(map[string]bool),
		TotalConnects:      0,
		store:              store,
		persistenceErr:     persistenceErr,
		adminKeys:          parseAdminUUIDs(os.Getenv("DMUD_ADMIN_UUIDS")),
		sessionKeys:        make(map[common.Client]string),
		autosaveInterval:   500 * time.Millisecond,
	}

	// Create day cycle system with broadcast callback
	dayCycleSystem := systems.NewDayCycleSystem(func(msg string) {
		game.Broadcast(msg)
	})
	world.AddSystem(dayCycleSystem)
	game.dayCycleSystem = dayCycleSystem

	// Give spawn system access to day cycle for night-only spawns
	spawnSystem.SetDayCycle(dayCycleSystem.GetDayCycle())

	game.initCommands()
	game.initializeSpawns()
	game.loadWorldState()

	go game.loop()

	return game
}

type spawnConfigJSON struct {
	TemplateID         string  `json:"template_id"`
	MinCount           int     `json:"min_count"`
	MaxCount           int     `json:"max_count"`
	RespawnTimeSeconds int     `json:"respawn_time_seconds"`
	Chance             float64 `json:"chance"`
	NightOnly          bool    `json:"night_only,omitempty"`
}

type areaSpawnJSON struct {
	AreaID string            `json:"area_id"`
	Spawns []spawnConfigJSON `json:"spawns"`
}

func (g *Game) initializeSpawns() {
	var areaSpawns []areaSpawnJSON
	if err := util.ParseJSON("./resources/spawns.json", &areaSpawns); err != nil {
		log.Error().Err(err).Msg("Failed to load spawns.json")
		return
	}

	for _, areaSpawn := range areaSpawns {
		var configs []components.SpawnConfig
		for _, s := range areaSpawn.Spawns {
			configs = append(configs, components.SpawnConfig{
				Type:        components.SpawnTypeNPC,
				TemplateID:  s.TemplateID,
				MinCount:    s.MinCount,
				MaxCount:    s.MaxCount,
				RespawnTime: time.Duration(s.RespawnTimeSeconds) * time.Second,
				Chance:      s.Chance,
				NightOnly:   s.NightOnly,
			})
		}

		spawn := components.NewSpawn(common.EntityID(areaSpawn.AreaID))
		spawn.Configs = configs

		entity, err := g.world.FindEntity(common.EntityID(areaSpawn.AreaID))
		if err == nil {
			g.world.AddComponent(&entity, spawn)
			log.Info().Msgf("Added spawn component to area %s with %d configs", areaSpawn.AreaID, len(configs))
		}
	}
}

func (g *Game) initCommands() {
	g.RegisterCommand(&Command{
		Name:        "look",
		Handler:     handleLook,
		Description: "Look around your current location.",
	})
	g.RegisterCommand(&Command{
		Name:        "who",
		Handler:     handleWho,
		Description: "List online players.",
	})
	g.RegisterCommand(&Command{
		Name:        "exit",
		Handler:     handleExit,
		Description: "Exit the game.",
	})
	g.RegisterCommand(&Command{
		Name:        "name",
		Handler:     handleName,
		Description: "Change your player name.",
	})
	g.RegisterCommand(&Command{
		Name:        "login",
		Handler:     handleLogin,
		Description: "Link this session to a UUID.",
	})
	g.RegisterCommand(&Command{
		Name:        "save",
		Handler:     handleSave,
		Description: "Save your character and get a UUID.",
	})
	g.RegisterCommand(&Command{
		Name:        "summon",
		Handler:     handleSummon,
		Description: "Summon a player to your location.",
	})
	g.RegisterCommand(&Command{
		Name:        "cast",
		Handler:     handleCast,
		Description: "Cast a spell.",
	})
	g.RegisterCommand(&Command{
		Name:        "adminstats",
		Handler:     handleAdminStats,
		Description: "Show admin entity counts.",
		Hidden:      true,
	})
	g.RegisterCommand(&Command{
		Name:        "recall",
		Handler:     handleRecall,
		Description: "Return to the starting area.",
	})
	g.RegisterCommand(&Command{
		Name:        "say",
		Handler:     handleSay,
		Description: "Say something to players in the same area.",
	})
	g.RegisterCommand(&Command{
		Name:        "shout",
		Handler:     handleShout,
		Description: "Shout a message to nearby players.",
	})
	g.RegisterCommand(&Command{
		Name:        "kill",
		Aliases:     []string{"k"},
		Handler:     handleKill,
		Description: "Attack another player or NPC.",
	})
	g.RegisterCommand(&Command{
		Name:        "examine",
		Aliases:     []string{"ex", "exa"},
		Handler:     handleExamine,
		Description: "Examine something or someone in detail.",
	})
	g.RegisterCommand(&Command{
		Name:        "time",
		Handler:     handleTime,
		Description: "Check the current time of day.",
	})
	g.RegisterCommand(&Command{
		Name:        "history",
		Aliases:     []string{"hist"},
		Handler:     handleHistory,
		Description: "Show your command history.",
	})
	g.RegisterCommand(&Command{
		Name:        "clear",
		Handler:     handleClear,
		Description: "Clear your command history.",
	})
	g.RegisterCommand(&Command{
		Name:        "suggest",
		Aliases:     []string{"sug"},
		Handler:     handleSuggest,
		Description: "Get suggestions for commands or player names.",
	})
	g.RegisterCommand(&Command{
		Name:        "complete",
		Aliases:     []string{"comp"},
		Handler:     handleComplete,
		Description: "Get instant auto-completion for commands or player names.",
	})
	g.RegisterCommand(&Command{
		Name:        "help",
		Aliases:     []string{"h", "?"},
		Handler:     handleHelp,
		Description: "Show help information for commands.",
	})
	g.RegisterCommand(&Command{
		Name:        "loot",
		Handler:     g.handleLoot,
		Description: "Loot items from a corpse.",
	})
	g.RegisterCommand(&Command{
		Name:        "inventory",
		Aliases:     []string{"inv", "i"},
		Handler:     g.handleInventory,
		Description: "View your inventory.",
	})
	g.RegisterCommand(&Command{
		Name:        "get",
		Aliases:     []string{"pickup", "take"},
		Handler:     g.handleGet,
		Description: "Pick up an item from the ground.",
	})
	g.RegisterCommand(&Command{
		Name:        "drop",
		Handler:     g.handleDrop,
		Description: "Drop an item from your inventory.",
	})
	g.RegisterCommand(&Command{
		Name:        "dropall",
		Handler:     g.handleDropAll,
		Description: "Drop all items (optionally matching a pattern).",
	})
	g.RegisterCommand(&Command{
		Name:        "sacrifice",
		Handler:     g.handleSacrifice,
		Aliases:     []string{"sac"},
		Description: "Destroy an item on the ground.",
	})
	g.RegisterCommand(&Command{
		Name:        "sacall",
		Handler:     g.handleSacrificeAll,
		Aliases:     []string{"sacrificeall"},
		Description: "Destroy all items on the ground (optionally matching a pattern).",
	})
	g.RegisterCommand(&Command{
		Name:        "hail",
		Handler:     g.handleHail,
		Description: "Hail an NPC to interact with them.",
	})
	g.RegisterCommand(&Command{
		Name:        "uptime",
		Handler:     handleUptime,
		Description: "Show server uptime and statistics.",
	})
	g.RegisterCommand(&Command{
		Name:    "xyzzy",
		Handler: handleXyzzy,
		Hidden:  true,
	})
	directions := map[string]string{
		"north": "n",
		"south": "s",
		"east":  "e",
		"west":  "w",
		"up":    "u",
		"down":  "d",
	}

	for dir, alias := range directions {
		g.RegisterCommand(&Command{
			Name:        dir,
			Aliases:     []string{alias},
			Handler:     g.createMoveHandler(dir),
			Description: "Move " + dir,
		})
	}
}

func (g *Game) RegisterCommand(cmd *Command) {
	commandRegistry[cmd.Name] = cmd
	for _, alias := range cmd.Aliases {
		commandRegistry[alias] = cmd
	}
}

func (g *Game) handleCommand(c ClientCommand) {
	client := c.Client

	cmdInput := c.Cmd
	cmdArgs := c.Args

	player, err := g.getPlayer(client)
	if err != nil {
		log.Warn().Msgf("Error getting player component: %s", err)
		return
	}

	// Add command to history
	fullCommand := cmdInput
	if len(cmdArgs) > 0 {
		fullCommand = cmdInput + " " + strings.Join(cmdArgs, " ")
	}
	player.CommandHistory.AddCommand(fullCommand)

	// Update auto-complete with all available commands
	for cmdName, cmd := range commandRegistry {
		if cmd.Hidden {
			continue
		}
		player.AutoComplete.AddCommand(cmdName)
	}

	// Update auto-complete with all player names
	g.playersMu.RLock()
	for playerName := range g.players {
		player.AutoComplete.AddPlayer(playerName)
	}
	g.playersMu.RUnlock()

	cmd, exists := commandRegistry[cmdInput]
	if exists {
		cmd.Handler(player, cmdArgs, g)
	} else {
		player.Broadcast(fmt.Sprintf("What do you mean, \"%s\"?", cmdInput))
	}

	// Send prompt after command is processed
	if client.SupportsPrompt() {
		client.SendMessage("> ")
	}
}

func (g *Game) HandleConnect(c common.Client) {
	deferSpawn := c.SupportsTags()

	loadedState, _ := g.loadPlayerState(c)
	playerName := util.GenerateRandomName()
	if loadedState != nil && strings.TrimSpace(loadedState.Name) != "" {
		if _, exists := g.players[loadedState.Name]; !exists {
			playerName = loadedState.Name
		}
	}

	playerArea := g.defaultArea
	if loadedState != nil {
		if area := g.resolveArea(loadedState.AreaID); area != nil {
			playerArea = area
		}
	}

	playerComponent := &components.Player{
		Client:         c,
		Name:           playerName,
		Area:           playerArea,
		CommandHistory: components.NewCommandHistory(),
		AutoComplete:   util.NewAutoComplete(),
		EnteredWorld:   !deferSpawn,
	}
	experienceComponent := components.NewExperience()
	healthComponent := components.NewHealth(experienceComponent.Level)
	inventoryComponent := components.NewInventory(0) // unlimited inventory
	questsComponent := components.NewPlayerQuests()

	playerEntity := ecs.NewEntity()
	g.world.AddEntity(playerEntity)

	g.world.AddComponent(&playerEntity, playerComponent)
	g.world.AddComponent(&playerEntity, experienceComponent)
	g.world.AddComponent(&playerEntity, healthComponent)
	g.world.AddComponent(&playerEntity, inventoryComponent)
	g.world.AddComponent(&playerEntity, questsComponent)

	if loadedState != nil {
		g.applyPlayerState(playerEntity.ID, playerComponent, loadedState)
	}

	g.playersMu.Lock()
	g.players[playerComponent.Name] = &playerEntity
	g.playersMu.Unlock()

	// Track connection stats
	g.TotalConnectMu.Lock()
	g.TotalConnects++
	g.TotalConnectMu.Unlock()

	// Track unique IPs (strip port from address)
	ipAddr := extractClientIP(c.RemoteAddr())

	g.UniqueIPsMu.Lock()
	g.UniqueIPs[ipAddr] = true
	g.UniqueIPsMu.Unlock()

	if !deferSpawn {
		playerComponent.Area.AddPlayer(playerComponent)

		playerComponent.Broadcast(util.WelcomeBanner)
		playerComponent.Look(g.world.AsWorldLike())
		playerComponent.BroadcastState(g.world.AsWorldLike(), playerEntity.ID)

		g.Broadcast(fmt.Sprintf("%s has joined the game.", playerComponent.Name), c)

		// Send initial prompt
		if c.SupportsPrompt() {
			c.SendMessage("> ")
		} else {
			c.SendMessage("\n") // spacer after the welcome text
		}
	} else {
		const loginGrace = 750 * time.Millisecond
		// Defer world entry briefly so a deferred-spawn (web) client can send
		// `login <uuid>` to retrieve a saved character first. The timer only
		// enqueues onto enterWorldChan; the actual state change happens on the
		// game loop in enterWorld, keeping all world mutation single-threaded.
		time.AfterFunc(loginGrace, func() {
			g.enterWorldChan <- c
		})
	}

	go c.HandleRequest()
}

// enterWorld completes a deferred spawn once the login grace elapses. It runs
// on the game loop goroutine (via enterWorldChan), serialized against commands
// and systems, so it requires no locking and cannot race the `login` command.
func (g *Game) enterWorld(c common.Client) {
	player, err := g.getPlayer(c)
	if err != nil {
		return // client disconnected during the login grace
	}

	if player.EnteredWorld {
		return // already entered the world (e.g. via `login` / character retrieval)
	}
	player.EnteredWorld = true
	if player.Area == nil {
		player.Area = g.defaultArea
	}

	player.Area.AddPlayer(player)

	player.Broadcast(util.WelcomeBanner)
	player.Look(g.world.AsWorldLike())
	if entityID, err := g.getPlayerEntity(player); err == nil {
		player.BroadcastState(g.world.AsWorldLike(), entityID)
	}

	g.Broadcast(util.TagMessage("STATUS", fmt.Sprintf("%s has joined the game.", player.Name)), c)
}

func (g *Game) HandleDisconnect(c common.Client) {
	player, err := g.getPlayer(c)
	if err != nil {
		return
	}

	g.playersMu.Lock()
	playerEntity := g.players[player.Name]
	if playerEntity == nil {
		g.playersMu.Unlock()
		log.Error().Msg("Player entity was nil")
		return
	}
	g.playersMu.Unlock()

	g.clearClientPersistenceKey(c)

	g.playersMu.Lock()
	g.world.RemoveEntity(playerEntity.ID)
	delete(g.players, player.Name)
	g.playersMu.Unlock()

	c.CloseConnection()
	g.Broadcast(util.TagMessage("STATUS", fmt.Sprintf("%s has left the game.", player.Name)), c)
}

func (g *Game) getPlayer(c common.Client) (*components.Player, error) {
	g.playersMu.RLock()
	defer g.playersMu.RUnlock()

	for _, playerEntity := range g.players {
		playerComponent, err := g.world.GetComponent(playerEntity.ID, "Player")
		if err != nil {
			return nil, fmt.Errorf("error getting player component for entity id %s, %v", playerEntity.ID, err)
		}
		player, ok := playerComponent.(*components.Player)
		if !ok {
			return nil, fmt.Errorf("unable to cast component to Player")
		}
		if player.Client == c {
			return player, nil
		}
	}
	return nil, fmt.Errorf("player not found")
}

func (g *Game) Broadcast(m string, excludeClients ...common.Client) {
	log.Info().Msgf("Broadcasting: %s", m)

	g.playersMu.RLock()
	defer g.playersMu.RUnlock()

	for _, playerEntity := range g.players {
		playerComponent, err := g.world.GetComponent(playerEntity.ID, "Player")
		if err != nil {
			log.Error().Msgf("Error getting player for entity id %s, %v", playerEntity.ID, err)
			continue
		}

		player, ok := playerComponent.(*components.Player)
		if !ok {
			log.Error().Msgf("Unable to cast component to Player %v", playerComponent)
			continue
		}

		if !util.ContainsClient(excludeClients, player.Client) {
			player.Broadcast(m)
		}
	}
}

func (g *Game) loop() {
	updateTicker := time.NewTicker(10 * time.Millisecond)
	defer updateTicker.Stop()

	for {
		select {
		case client := <-g.AddPlayerChan:
			g.HandleConnect(client)
		case client := <-g.RemovePlayerChan:
			g.HandleDisconnect(client)
		case client := <-g.enterWorldChan:
			g.enterWorld(client)
		case command := <-g.ExecuteCommandChan:
			g.handleCommand(command)
		case <-updateTicker.C:
			g.world.Update()
			g.autosaveTick()
		}
	}
}

func parseAdminUUIDs(value string) map[string]bool {
	result := make(map[string]bool)
	for _, raw := range strings.Split(value, ",") {
		candidate := strings.TrimSpace(raw)
		if candidate == "" {
			continue
		}
		result[candidate] = true
	}
	return result
}

func (g *Game) isAdminUUID(uuid string) bool {
	uuid = strings.TrimSpace(uuid)
	if uuid == "" {
		return false
	}
	return g.adminKeys[uuid]
}

func (g *Game) resolveAdminStatus(uuid string) bool {
	if g.isAdminUUID(uuid) {
		return true
	}
	if !g.persistenceEnabled() {
		return false
	}
	isAdmin, err := g.store.IsAdmin(context.Background(), uuid)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to check admin flag")
		return false
	}
	return isAdmin
}

func (g *Game) countByComponent(componentName string) int {
	entities, err := g.world.FindEntitiesByComponentPredicate(componentName, func(i interface{}) bool {
		return true
	})
	if err != nil {
		return 0
	}
	return len(entities)
}

func (g *Game) countRoomsAndExits() (int, int) {
	areas, err := g.world.FindEntitiesByComponentPredicate("Area", func(i interface{}) bool {
		return true
	})
	if err != nil {
		return 0, 0
	}
	exitCount := 0
	for _, entity := range areas {
		areaComp, err := g.world.GetComponent(entity.ID, "Area")
		if err != nil {
			continue
		}
		area, ok := areaComp.(*components.Area)
		if !ok || area == nil {
			continue
		}
		exitCount += len(area.Exits)
	}
	return len(areas), exitCount
}
