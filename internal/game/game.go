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

///////////////////////////////////////////////////////////////////////////////////////////////
// Game
//

type Game struct {
	defaultRoom *components.RoomComponent

	players   map[string]*components.PlayerComponent
	playersMu sync.Mutex

	world *ecs.World

	AddPlayerChan    chan common.Client
	RemovePlayerChan chan common.Client
	CommandChan      chan ClientCommand
}

///////////////////////////////////////////////////////////////////////////////////////////////
// ..
//

func (g *Game) HandleConnect(c common.Client) {
	playerComponent := &components.PlayerComponent{
		Client: c,
		Name:   util.GenerateRandomName(),
		Room:   g.defaultRoom,
	}
	healthComponent := &components.HealthComponent{
		MaxHealth:     100,
		CurrentHealth: 100,
	}
	g.defaultRoom.AddPlayer(playerComponent)

	g.playersMu.Lock()

	g.players[playerComponent.Name] = playerComponent

	playerEntity := ecs.NewEntity()
	g.world.AddEntity(playerEntity)

	g.world.AddComponent(playerEntity, playerComponent)
	g.world.AddComponent(playerEntity, healthComponent)

	g.playersMu.Unlock()

	g.Broadcast(fmt.Sprintf("%s has joined the game.", playerComponent.Name), c)

	c.SendMessage(util.WelcomeBanner)
	c.SendMessage(g.defaultRoom.Description)

	log.Info().Msg(fmt.Sprintf("Player %s added", playerComponent.Name))

	go c.HandleRequest()
}

func (g *Game) HandleDisconnect(c common.Client) {
	g.playersMu.Lock()

	player, err := g.getPlayer(c)
	if err != nil {
		log.Error().Err(err).Msg("Error getting disconnected player")
		return
	}

	delete(g.players, player.Name)
	g.playersMu.Unlock()

	if err := c.CloseConnection(); err != nil {
		log.Error().Err(err).Msg("Error closing client connection")
		return
	}

	g.Broadcast(fmt.Sprintf("%s has left the game.", player.Name), c)
}

///////////////////////////////////////////////////////////////////////////////////////////////
// ..
//

func (g *Game) getPlayer(c common.Client) (*components.PlayerComponent, error) {
	for _, player := range g.players {
		if player.Client == c {
			return player, nil
		}
	}
	return nil, fmt.Errorf("player not found")
}

func (g *Game) handleCommand(c ClientCommand) {
	command := c.Command
	client := c.Client

	player, err := g.getPlayer(client)
	if err != nil {
		log.Warn().Msg(fmt.Sprintf("Error getting player component: %s", err))
		return
	}

	dirMapping := map[string]string{
		"n": "north",
		"s": "south",
		"e": "east",
		"w": "west",
		"u": "up",
		"d": "down",
	}

	switch c.Command.Cmd {
	case "exit":
		g.handleExit(player, command)
	case "kill":
		go player.Kill(command.Args[1])
	case "look":
		go player.Look()
	case "n", "s", "e", "w", "u", "d", "north", "south", "east", "west", "up", "down":
		fullDir := c.Command.Cmd
		if shortDir, ok := dirMapping[c.Command.Cmd]; ok {
			fullDir = shortDir
		}
		go player.Move(fullDir)
	case "scan":
		go player.Scan()
	case "say":
		go player.Say(strings.Join(command.Args, " "))
	case "shout":
		go player.Shout(strings.Join(command.Args, " "))
	default:
		client.SendMessage(fmt.Sprintf("What do you mean, \"%s\"?", command.Cmd))
	}
}

func (g *Game) handleExit(player *components.PlayerComponent, command Command) {
	player.Client.CloseConnection()
	g.Broadcast(fmt.Sprintf("%s has left the game.", player.Name), player.Client)
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
		case command := <-g.CommandChan:
			g.handleCommand(command)
		case <-updateTicker.C:
			g.world.Update()
		}
	}
}

func (g *Game) Broadcast(m string, excludeClients ...common.Client) {
	g.playersMu.Lock()
	defer g.playersMu.Unlock()

	for _, player := range g.players {
		if !util.ContainsClient(excludeClients, player.Client) {
			player.Client.SendMessage(m)
		}
	}
}

///////////////////////////////////////////////////////////////////////////////////////////////
// ..
//

func NewGame() *Game {
	world := ecs.NewWorld()
	defaultRoom, _ := world.GetComponent("1", "RoomComponent")
	game := Game{
		defaultRoom:      defaultRoom.(*components.RoomComponent),
		players:          make(map[string]*components.PlayerComponent),
		world:            world,
		AddPlayerChan:    make(chan common.Client),
		RemovePlayerChan: make(chan common.Client),
		CommandChan:      make(chan ClientCommand),
	}

	combatSytem := &systems.CombatSystem{}
	world.AddSystem(combatSytem)
	go game.loop()
	return &game
}
