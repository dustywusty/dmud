package game

import (
	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/net"
	"time"
)

type Game struct {
	world            *ecs.World
	AddPlayerChan    chan *net.Client
	RemovePlayerChan chan *net.Client
}

func NewGame() *Game {
	game := &Game{
		world:            ecs.NewWorld(),
		AddPlayerChan:    make(chan *net.Client),
		RemovePlayerChan: make(chan *net.Client),
	}
	go game.loop()
	return game
}

func (g *Game) AddPlayer(c *net.Client) {
	playerEntity := ecs.Entity{}
	playerComponent := &components.PlayerComponent{
		Client: c,
	}
	g.world.AddComponent(playerEntity, playerComponent)
}

func (g *Game) RemovePlayer(c *net.Client) {
	// Find the player entity associated with the given client
	playerEntity, err := g.world.FindEntityByComponentPredicate("PlayerComponent", func(component interface{}) bool {
		if playerComponent, ok := component.(*components.PlayerComponent); ok {
			return playerComponent.Client == c
		}
		return false
	})
	if err {
		return
	}
	g.world.RemoveEntity(playerEntity.ID)
}

func (g *Game) loop() {
	for {
		select {
		case player := <-g.AddPlayerChan:
			g.AddPlayer(player)
		case player := <-g.RemovePlayerChan:
			g.RemovePlayer(player)
		default:
			g.world.Update()
			time.Sleep(10 * time.Millisecond)
		}
	}
}
