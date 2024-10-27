package game

import (
	"fmt"
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

// CommandHandler is a function type for handling commands.
type CommandHandler func(player *components.Player, args []string, game *Game)

// Command represents a game command with its handler and metadata.
type Command struct {
	Name        string
	Aliases     []string
	Handler     CommandHandler
	Description string
}

// commandRegistry maps command names to Command structs.
var commandRegistry = make(map[string]*Command)

// Game represents the game state and contains all game-related methods.
type Game struct {
	defaultRoom *components.Room

	players   map[string]*ecs.Entity
	playersMu sync.Mutex

	world *ecs.World

	AddPlayerChan      chan common.Client
	RemovePlayerChan   chan common.Client
	ExecuteCommandChan chan ClientCommand
}

// NewGame initializes a new Game instance.
func NewGame() *Game {
	combatSystem := &systems.CombatSystem{}
	movementSystem := &systems.MovementSystem{}

	world := ecs.NewWorld()
	world.AddSystem(combatSystem)
	world.AddSystem(movementSystem)

	defaultRoomUntyped, err := world.GetComponent("1", "Room")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get default room")
	}

	defaultRoom, ok := defaultRoomUntyped.(*components.Room)
	if !ok {
		log.Fatal().Msg("Failed to cast default room to *components.Room")
	}

	game := &Game{
		defaultRoom:        defaultRoom,
		players:            make(map[string]*ecs.Entity),
		world:              world,
		AddPlayerChan:      make(chan common.Client),
		RemovePlayerChan:   make(chan common.Client),
		ExecuteCommandChan: make(chan ClientCommand),
	}

	game.initCommands()

	go game.loop()

	return game
}

// initCommands registers all game commands.
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
		Name:        "say",
		Handler:     handleSay,
		Description: "Say something to players in the same room.",
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

	// Movement commands
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

// RegisterCommand adds a command and its aliases to the registry.
func (g *Game) RegisterCommand(cmd *Command) {
	commandRegistry[cmd.Name] = cmd
	for _, alias := range cmd.Aliases {
		commandRegistry[alias] = cmd
	}
}

// handleCommand processes a command issued by a client.
func (g *Game) handleCommand(c ClientCommand) {
	client := c.Client

	cmdInput := c.Cmd
	cmdArgs := c.Args

	player, err := g.getPlayer(client)
	if err != nil {
		log.Warn().Msgf("Error getting player component: %s", err)
		return
	}

	cmd, exists := commandRegistry[cmdInput]
	if exists {
		cmd.Handler(player, cmdArgs, g)
	} else {
		player.Broadcast(fmt.Sprintf("What do you mean, \"%s\"?", cmdInput))
	}
}

// HandleConnect is called when a new client connects to the game.
func (g *Game) HandleConnect(c common.Client) {
	playerComponent := &components.Player{
		Client: c,
		Name:   util.GenerateRandomName(),
		Room:   g.defaultRoom,
	}
	healthComponent := &components.Health{
		Max:     100,
		Current: 100,
	}

	playerEntity := ecs.NewEntity()
	g.world.AddEntity(playerEntity)

	g.world.AddComponent(&playerEntity, playerComponent)
	g.world.AddComponent(&playerEntity, healthComponent)

	g.playersMu.Lock()
	g.players[playerComponent.Name] = &playerEntity
	g.playersMu.Unlock()

	g.defaultRoom.AddPlayer(playerComponent)

	playerComponent.Broadcast(util.WelcomeBanner)
	playerComponent.Broadcast(g.defaultRoom.Description)

	g.Broadcast(fmt.Sprintf("%s has joined the game.", playerComponent.Name), c)

	go c.HandleRequest()
}

// HandleDisconnect is called when a client disconnects from the game.
func (g *Game) HandleDisconnect(c common.Client) {
	player, err := g.getPlayer(c)
	if err != nil {
		return
	}

	g.playersMu.Lock()
	playerEntity := g.players[player.Name]
	if playerEntity == nil {
		log.Error().Msg("Player entity was nil")
		g.playersMu.Unlock()
		return
	}

	g.world.RemoveEntity(playerEntity.ID)
	delete(g.players, player.Name)
	g.playersMu.Unlock()

	c.CloseConnection()

	g.Broadcast(fmt.Sprintf("%s has left the game.", player.Name), c)
}

// getPlayer retrieves the Player component associated with a client.
func (g *Game) getPlayer(c common.Client) (*components.Player, error) {
	g.playersMu.Lock()
	defer g.playersMu.Unlock()

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

// Broadcast sends a message to all players, excluding specified clients.
func (g *Game) Broadcast(m string, excludeClients ...common.Client) {
	log.Info().Msgf("Broadcasting: %s", m)

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

// loop is the main game loop that processes player connections and updates the world.
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
