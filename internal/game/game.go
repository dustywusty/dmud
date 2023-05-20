package game

import (
	"log"
	"time"

	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
)

type Client interface{}

type Game struct {
	World            *ecs.World
	AddPlayerChan    chan common.Client
	RemovePlayerChan chan common.Client
}

func NewGame() *Game {
	game := &Game{
		World:            ecs.NewWorld(),
		AddPlayerChan:    make(chan common.Client),
		RemovePlayerChan: make(chan common.Client),
	}
	go game.loop()
	return game
}

func (g *Game) AddPlayer(c common.Client) {
	playerEntity := ecs.NewEntity()
	playerComponent := components.PlayerComponent{
		Client: c,
	}

	g.World.AddEntity(playerEntity)
	g.World.AddComponent(playerEntity, &playerComponent)

	log.Printf("Adding player %v", string(playerEntity.ID))
}

func (g *Game) RemovePlayer(c common.Client) {
	log.Printf("Attempting to remove player %v", c)

	playerEntity, err := g.World.FindEntityByComponentPredicate("PlayerComponent", func(component interface{}) bool {
		if playerComponent, ok := component.(*components.PlayerComponent); ok {
			log.Printf("Found player component: %v", playerComponent.Client)
			return playerComponent.Client == c
		}
		return false
	})
	if err != nil {
		log.Printf("Error removing player: %v", err)
		return
	}
	log.Printf("Removing player %v", playerEntity.ID)
	g.World.RemoveEntity(playerEntity.ID)
}

func (g *Game) loop() {
	for {
		select {
		case client := <-g.AddPlayerChan:
			g.AddPlayer(client)
		case client := <-g.RemovePlayerChan:
			g.RemovePlayer(client)
		default:
			g.World.Update()
			time.Sleep(10 * time.Millisecond)
		}
	}
}
