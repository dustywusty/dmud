package game

import (
	"fmt"
	"log"
	"strings"
	"time"

	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/util"
)

///////////////////////////////////////////////////////////////////////////////////////////////
// Game
//

type Game struct {
	defaultRoom *components.RoomComponent
	players     []ecs.Component
	world       *ecs.World

	AddPlayerChan    chan common.Client
	RemovePlayerChan chan common.Client

	CommandChan chan ClientCommand
}

func NewGame() *Game {
	world := ecs.NewWorld()
	defaultRoom, _ := world.GetComponent("1", "RoomComponent")

	game := Game{
		defaultRoom:      defaultRoom.(*components.RoomComponent),
		players:          make([]ecs.Component, 0),
		world:            world,
		AddPlayerChan:    make(chan common.Client),
		RemovePlayerChan: make(chan common.Client),
		CommandChan:      make(chan ClientCommand),
	}

	go game.loop()

	return &game
}

///////////////////////////////////////////////////////////////////////////////////////////////
// Public
//

func (g *Game) AddPlayer(c common.Client) {
	playerComponent := components.PlayerComponent{
		Client: c,
		Name:   util.GenerateRandomName(),
		Room:   g.defaultRoom,
	}
	g.players = append(g.players, &playerComponent)

	playerEntity := ecs.NewEntity()

	g.world.AddEntity(playerEntity)
	g.world.AddComponent(playerEntity, &playerComponent)

	g.messageAllPlayers(fmt.Sprintf("%v has joined the game.", playerComponent.Name), c)

	// enter the room

	log.Printf("Adding player %v", string(playerComponent.Name))
}

func (g *Game) HandleDisconnect(c common.Client) {
	g.RemovePlayerChan <- c
}

func (g *Game) RemovePlayer(c common.Client) {
	playerEntity, err := g.world.FindEntityByComponentPredicate("PlayerComponent", func(component interface{}) bool {
		if playerComponent, ok := component.(*components.PlayerComponent); ok {
			return playerComponent.Client == c
		}
		return false
	})
	if err != nil {
		log.Printf("Error removing player: %v", err)
		return
	}
	playerComponent, err := g.world.GetComponent(playerEntity.ID, "PlayerComponent")
	if err != nil {
		log.Printf("Error getting player component: %v", err)
		return
	}
	log.Printf("Removing player %v", playerComponent.(*components.PlayerComponent).Name)
	g.world.RemoveEntity(playerEntity.ID)
}

///////////////////////////////////////////////////////////////////////////////////////////////
// Private
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
		if player.(*components.PlayerComponent).Client == c {
			return player.(*components.PlayerComponent), nil
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
	player.Client.SendMessage("What?")
}

func (g *Game) loop() {
	for {
		select {
		case client := <-g.AddPlayerChan:
			g.AddPlayer(client)
		case client := <-g.RemovePlayerChan:
			g.RemovePlayer(client)
		case clientCommand := <-g.CommandChan:
			g.handleCommand(clientCommand)
		default:
			g.world.Update()
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (g *Game) messageAllPlayers(m string, excludeClients ...common.Client) {
	for _, player := range g.players {
		if !containsClient(excludeClients, player.(*components.PlayerComponent).Client) {
			player.(*components.PlayerComponent).Client.SendMessage(m)
		}
	}
}
