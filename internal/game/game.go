package game

import (
	"log"
	"time"

	"dmud/internal/components"
	"dmud/internal/ecs"
)

type Client interface{}

type Game struct {
	World            *ecs.World
	AddPlayerChan    chan *ecs.Entity
	RemovePlayerChan chan *ecs.Entity
}

func NewGame() *Game {
	game := &Game{
		World:            ecs.NewWorld(),
		AddPlayerChan:    make(chan *ecs.Entity),
		RemovePlayerChan: make(chan *ecs.Entity),
	}
	go game.loop()
	return game
}

func (g *Game) AddPlayer(c Client) {
	playerEntity := ecs.NewEntity()
	playerComponent := components.PlayerComponent{
		Client: c,
	}

	g.World.AddEntity(playerEntity)
	g.World.AddComponent(playerEntity, &playerComponent)

	log.Printf("Adding player %v", string(playerEntity.ID))
}

func (g *Game) RemovePlayer(c *Client) {
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
	g.World.RemoveEntity(playerEntity)
	log.Printf("Removing player %v", string(playerEntity.ID))
}

func (g *Game) loop() {
	for {
		select {
		case player := <-g.AddPlayerChan:
			g.AddPlayer(player.ID)
		case player := <-g.RemovePlayerChan:
			g.RemovePlayer(player)
		default:
			g.World.Update()
			time.Sleep(10 * time.Millisecond)
		}
	}
}
