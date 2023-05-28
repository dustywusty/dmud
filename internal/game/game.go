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
	World *ecs.World

	AddPlayerChan    chan common.Client
	RemovePlayerChan chan common.Client

	CommandChan chan ClientCommand
}

func (g *Game) AddPlayer(c common.Client) {
	defaultRoomComponent, err := g.World.GetComponent("Room1", "RoomComponent")
	if err != nil {
		log.Printf("Default room does not have a RoomComponent")
		return
	}

	roomComponent, ok := defaultRoomComponent.(*components.RoomComponent)
	if !ok {
		log.Printf("Default room does not have a RoomComponent")
		return
	}

	playerComponent := components.PlayerComponent{
		Client: c,
		Name:   util.GenerateRandomName(),
		Room:   roomComponent,
	}

	playerEntity := ecs.NewEntity()

	g.World.AddEntity(playerEntity)
	g.World.AddComponent(playerEntity, &playerComponent)

	g.messageAllPlayers(fmt.Sprintf("%v has joined the game.", playerComponent.Name), c)

	c.SendMessage(util.WelcomeBanner)
	c.SendMessage(roomComponent.Description)

	log.Printf("Adding player %v", string(playerComponent.Name))
}

func (g *Game) RemovePlayer(c common.Client) {
	playerEntity, err := g.World.FindEntityByComponentPredicate("PlayerComponent", func(component interface{}) bool {
		if playerComponent, ok := component.(*components.PlayerComponent); ok {
			return playerComponent.Client == c
		}
		return false
	})
	if err != nil {
		log.Printf("Error removing player: %v", err)
		return
	}
	playerComponent, err := g.World.GetComponent(playerEntity.ID, "PlayerComponent")
	if err != nil {
		log.Printf("Error getting player component: %v", err)
		return
	}
	log.Printf("Removing player %v", playerComponent.(*components.PlayerComponent).Name)
	g.World.RemoveEntity(playerEntity.ID)
}

///////////////////////////////////////////////////////////////////////////////////////////////
// Game Loop
//

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
			g.World.Update()
			time.Sleep(10 * time.Millisecond)
		}
	}
}

///////////////////////////////////////////////////////////////////////////////////////////////
// Helpers
//

func containsClient(clients []common.Client, client common.Client) bool {
	for _, c := range clients {
		if c == client {
			return true
		}
	}
	return false
}

func (g *Game) getPlayerComponent(c common.Client) (*components.PlayerComponent, error) {
	playerEntity, err := g.World.FindEntityByComponentPredicate("PlayerComponent", func(component interface{}) bool {
		if playerComponent, ok := component.(*components.PlayerComponent); ok {
			return playerComponent.Client == c
		}
		return false
	})
	if err != nil {
		return nil, err
	}
	playerComponent, err := g.World.GetComponent(playerEntity.ID, "PlayerComponent")
	if err != nil {
		return nil, err
	}
	return playerComponent.(*components.PlayerComponent), nil
}

func (g *Game) messageAllPlayers(m string, excludeClients ...common.Client) {
	players, _ := g.World.FindEntitiesByComponentPredicate("PlayerComponent", func(c interface{}) bool {
		_, ok := c.(*components.PlayerComponent)
		return ok
	})

	for _, player := range players {
		playerComponent, err := g.World.GetComponent(player.ID, "PlayerComponent")
		if err != nil {
			fmt.Println("Error getting PlayerComponent:", err)
			continue
		}
		if playerComp, ok := playerComponent.(*components.PlayerComponent); ok {
			if playerComp.Client != nil && !containsClient(excludeClients, playerComp.Client) {
				playerComp.Client.SendMessage(m)
			}
		}
	}
}

///////////////////////////////////////////////////////////////////////////////////////////////
// Commands
//

func (g *Game) handleCommand(c ClientCommand) {
	client := c.Client
	player, err := g.getPlayerComponent(client)
	if err != nil {
		fmt.Println("Error getting player component:", err)
		return
	}
	switch c.Command.Cmd {
	case "exit":
		g.handleExit(&c.Command, player)
	case "shout":
		g.handleShout(&c.Command, player)
	default:
		g.handleUnknownCommand(&c.Command, player)
	}
}

func (g *Game) handleExit(command *Command, player *components.PlayerComponent) {
	player.Client.CloseConnection()
	g.messageAllPlayers(fmt.Sprintf("%s has left the game.", player.Name), player.Client)
}

func (g *Game) handleShout(command *Command, player *components.PlayerComponent) {
	message := fmt.Sprintf("%s shouts %s", player.Name, strings.Join(command.Args, " "))
	g.messageAllPlayers(message, player.Client)
	player.Client.SendMessage(fmt.Sprintf("You shout %s", strings.Join(command.Args, " ")))
}

func (g *Game) handleUnknownCommand(command *Command, player *components.PlayerComponent) {
	player.Client.SendMessage("What?")
}

///////////////////////////////////////////////////////////////////////////////////////////////
// New game helper
//

func NewGame() *Game {
	game := &Game{
		World:            ecs.NewWorld(),
		AddPlayerChan:    make(chan common.Client),
		RemovePlayerChan: make(chan common.Client),
		CommandChan:      make(chan ClientCommand),
	}

	go game.loop()

	return game
}
