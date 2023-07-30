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

func (g *Game) HandleConnect(c common.Client) {

}

func (g *Game) HandleDisconnect(c common.Client) {
	g.RemovePlayerChan <- c
}

func (g *Game) RemovePlayer(c common.Client) {
	g.playersMu.Lock()
	defer g.playersMu.Unlock()

	player, err := g.getPlayer(c)
	if err != nil {
		log.Error().Err(err).Msg("")
		return
	}

	// g.world.RemoveEntity(player.EntityID)
	delete(g.players, player.Name)

	g.messageAllPlayers(fmt.Sprintf("%s has left the game.", player.Name), c)
}

func (g *Game) loop() {
	updateTicker := time.NewTicker(10 * time.Millisecond)
	defer updateTicker.Stop()

	for {
		select {
		case client := <-g.AddPlayerChan:
			g.HandleConnect(client)
		case client := <-g.RemovePlayerChan:
			g.RemovePlayer(client)
		case command := <-g.CommandChan:
			g.handleCommand(command)
		case <-updateTicker.C:
			g.world.Update()
		}
	}
}

func (g *Game) addPlayer(p *components.PlayerComponent) {
	g.playersMu.Lock()
	defer g.playersMu.Unlock()

	g.players[p.Name] = p

	playerEntity := ecs.NewEntity()
	g.world.AddEntity(playerEntity)
	g.world.AddComponent(playerEntity, p)

	g.messageAllPlayers(fmt.Sprintf("%s has joined the game.", p.Name), p.Client)

	p.Client.SendMessage(util.WelcomeBanner)
	p.Client.SendMessage(g.defaultRoom.Description)

	go p.Client.HandleRequest()

	log.Info().Msg(fmt.Sprintf("Player %s added", p.Name))
}

func containsClient(clients []common.Client, client common.Client) bool {
	for _, c := range clients {
		if c == client {
			return true
		}
	}
	return false
}

func (g *Game) getPlayer(c common.Client) (*components.PlayerComponent, error) {
	g.playersMu.Lock()
	defer g.playersMu.Unlock()

	for _, player := range g.players {
		if player.Client == c {
			return player, nil
		}
	}
	return nil, fmt.Errorf("Player not found")
}

func (g *Game) handleCommand(c ClientCommand) {
	command := c.Command
	client := c.Client
	player, err := g.getPlayer(client)

	if err != nil {
		fmt.Println("Error getting player component:", err)
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

func (g *Game) messageAllPlayers(m string, excludeClients ...common.Client) {
	g.playersMu.Lock()
	defer g.playersMu.Unlock()

	for _, player := range g.players {
		if !containsClient(excludeClients, player.Client) {
			player.Client.SendMessage(m)
		}
	}
}
