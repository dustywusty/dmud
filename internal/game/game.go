package game

import (
	"dmud/internal/ecs"
	"log"
	"net"
	"time"
)

type Game struct {
	World *ecs.World
}

func NewGame() *Game {
	game := &Game{
		World: ecs.NewWorld(),
	}

	go game.loop()

	return game
}

func (g *Game) AddNewPlayer(conn net.Conn) {
	// Create a new player entity
	player := g.World.CreateEntity()
	g.World.AddComponent(player, &ecs.Player{Conn: conn})

	// Set up the connection listener for the player
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				log.Printf("Error reading from connection: %v", err)
				break
			}
			// Process the received data
			data := string(buf[:n])
			// You may want to create a new command and add it to the World's CommandChan
			// based on the received data
		}
	}()
}

func calculateDeltaTime() float64 {
	if lastTime.IsZero() {
		lastTime = time.Now()
		return 0
	}
	currentTime := time.Now()
	deltaTime := currentTime.Sub(lastTime).Seconds()
	lastTime = currentTime
	return deltaTime
}

func (g *Game) loop() {
	for {
		select {
		case player := <-g.World.AddPlayerChan:
			g.World.AddPlayer(player)
		case player := <-g.World.RemovePlayerChan:
			g.World.RemovePlayer(player)
		case playerCmd := <-g.World.CommandChan:
			g.ExecuteCommand(playerCmd.player, playerCmd.command)
		default:
			dt := calculateDeltaTime()
			g.World.Update(dt)
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// Implement the command execution logic in the ECS systems, not in the game loop
func (g *Game) ExecuteCommand(player ecs.Entity, cmd *ecs.Command) {
	// Add the command execution logic to your ECS systems
}
