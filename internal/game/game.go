package game

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
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

	g.playersMu.Lock()
	g.players[playerComponent.Name] = playerComponent
	playerEntity := ecs.NewEntity()
	g.world.AddEntity(playerEntity)
	g.world.AddComponent(playerEntity, playerComponent)
	g.playersMu.Unlock()

	g.messageAllPlayers(fmt.Sprintf("%s has joined the game.", playerComponent.Name), c)

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

	g.messageAllPlayers(fmt.Sprintf("%s has left the game.", player.Name), c)
}

///////////////////////////////////////////////////////////////////////////////////////////////
// ..
//

func containsClient(clients []common.Client, client common.Client) bool {
	for _, c := range clients {
		if c == client {
			return true
		}
	}
	return false
}

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

	switch c.Command.Cmd {
	case "exit":
		g.handleExit(player, command)
	case "shout":
		g.handleShout(player, command)
	default:
		g.handleUnknownCommand(player, command)
	}
}

func (g *Game) handleExit(player *components.PlayerComponent, command Command) {
	player.Client.CloseConnection()

	g.messageAllPlayers(fmt.Sprintf("%s has left the game.", player.Name), player.Client)
}

func (g *Game) handleShout(player *components.PlayerComponent, command Command) {
	message := fmt.Sprintf("%s shouts %s", player.Name, strings.Join(command.Args, " "))
	g.messageAllPlayers(message, player.Client)
	player.Client.SendMessage(fmt.Sprintf("You shout %s", strings.Join(command.Args, " ")))
}

func (g *Game) handleUnknownCommand(player *components.PlayerComponent, command Command) {
	player.Client.SendMessage(fmt.Sprintf("What do you mean, \"%s\"?", command.Cmd))
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

func (g *Game) messageAllPlayers(m string, excludeClients ...common.Client) {
	g.playersMu.Lock()
	defer g.playersMu.Unlock()

	for _, player := range g.players {
		if !containsClient(excludeClients, player.Client) {
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

	go game.loop()

	return &game
}
