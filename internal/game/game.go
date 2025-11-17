package game

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
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

	AddPlayerChan      chan common.Client
	RemovePlayerChan   chan common.Client
	ExecuteCommandChan chan ClientCommand
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

	game := &Game{
		defaultArea:        defaultArea,
		players:            make(map[string]*ecs.Entity),
		world:              world,
		AddPlayerChan:      make(chan common.Client, 64),
		RemovePlayerChan:   make(chan common.Client, 64),
		ExecuteCommandChan: make(chan ClientCommand, 256),
	}

	game.initCommands()
	game.initializeSpawns()

	go game.loop()

	return game
}

func (g *Game) initializeSpawns() {
	// Add spawns to specific areas
	spawns := map[string][]components.SpawnConfig{
		"1": { // Starting area
			{
				Type:        components.SpawnTypeNPC,
				TemplateID:  "rat",
				MinCount:    5,
				MaxCount:    10,
				RespawnTime: 30 * time.Second,
				Chance:      1,
			},
		},
		"2": { // Another area
			{
				Type:        components.SpawnTypeNPC,
				TemplateID:  "goblin",
				MinCount:    1,
				MaxCount:    2,
				RespawnTime: 60 * time.Second,
				Chance:      0.6,
			},
			{
				Type:        components.SpawnTypeNPC,
				TemplateID:  "rat",
				MinCount:    2,
				MaxCount:    4,
				RespawnTime: 30 * time.Second,
				Chance:      0.9,
			},
		},
		"3": { // Town square
			{
				Type:        components.SpawnTypeNPC,
				TemplateID:  "guard",
				MinCount:    2,
				MaxCount:    2,
				RespawnTime: 120 * time.Second,
				Chance:      1.0,
			},
			{
				Type:        components.SpawnTypeNPC,
				TemplateID:  "merchant",
				MinCount:    1,
				MaxCount:    1,
				RespawnTime: 180 * time.Second,
				Chance:      1.0,
			},
		},
		"100": { // Chicken farm west
			{
				Type:        components.SpawnTypeNPC,
				TemplateID:  "chicken",
				MinCount:    20,
				MaxCount:    20,
				RespawnTime: 5 * time.Second,
				Chance:      1.0,
			},
		},
		"101": { // Chicken farm east
			{
				Type:        components.SpawnTypeNPC,
				TemplateID:  "chicken",
				MinCount:    20,
				MaxCount:    20,
				RespawnTime: 5 * time.Second,
				Chance:      1.0,
			},
		},
	}

	for areaID, configs := range spawns {
		spawn := components.NewSpawn(common.EntityID(areaID))
		spawn.Configs = configs

		entity, err := g.world.FindEntity(common.EntityID(areaID))
		if err == nil {
			g.world.AddComponent(&entity, spawn)
			log.Info().Msgf("Added spawn component to area %s with %d configs", areaID, len(configs))
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
	playerComponent := &components.Player{
		Client:         c,
		Name:           util.GenerateRandomName(),
		Area:           g.defaultArea,
		CommandHistory: components.NewCommandHistory(),
		AutoComplete:   util.NewAutoComplete(),
	}
	experienceComponent := components.NewExperience()
	healthComponent := components.NewHealth(experienceComponent.Level)

	playerEntity := ecs.NewEntity()
	g.world.AddEntity(playerEntity)

	g.world.AddComponent(&playerEntity, playerComponent)
	g.world.AddComponent(&playerEntity, experienceComponent)
	g.world.AddComponent(&playerEntity, healthComponent)

	g.playersMu.Lock()
	g.players[playerComponent.Name] = &playerEntity
	g.playersMu.Unlock()

	g.defaultArea.AddPlayer(playerComponent)

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

	go c.HandleRequest()
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
	g.world.RemoveEntity(playerEntity.ID)
	delete(g.players, player.Name)
	g.playersMu.Unlock()

	c.CloseConnection()
	g.Broadcast(fmt.Sprintf("%s has left the game.", player.Name), c)
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
		case command := <-g.ExecuteCommandChan:
			g.handleCommand(command)
		case <-updateTicker.C:
			g.world.Update()
		}
	}
}
