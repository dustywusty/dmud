package game

import (
	"dmud/internal/ecs"
	"fmt"
	"log"
	"strings"
	"time"
)

type Game struct {
	clients          map[string]Player
	addPlayerChan    chan Player
	removePlayerChan chan Player
	commandChan      chan *PlayerCommand
	rooms            map[int]*Room
	systems          []ecs.System
	entities         []ecs.Entity
}

type PlayerCommand struct {
	player  Player
	command *Command
}

func (g *Game) AddPlayer(player Player) {
	g.addPlayerChan <- player
	startingRoom := g.rooms[1]
	startingRoom.AddPlayer(player)
	player.SendMessage(startingRoom.Description)
}

func (g *Game) ExecuteCommand(player Player, cmd *Command) {
	switch cmd.Name {
	case "shout":
		g.handleShout(player, cmd)
	case "look":
		g.handleLook(player)
	case "say":
		g.handleSay(player, cmd)
	case "scan":
		g.handleScan(player)
	case "north", "south", "east", "west", "up", "down":
		g.handleMove(player, cmd.Name)
	case "exit":
		g.handleExit(player)
	default:
		player.SendMessage("Huh?")
	}
}

func NewGame() *Game {
	game := &Game{
		clients:          make(map[string]Player),
		addPlayerChan:    make(chan Player),
		removePlayerChan: make(chan Player),
		commandChan:      make(chan *PlayerCommand),
		rooms:            make(map[int]*Room),
	}

	room1 := NewRoom(1, "Room 1", "The room is white and nondescript, with claw marks on the ceiling that you can't help but notice. In one corner, there's a pile of Ayn Rand books.\n\nThe atmosphere is oppressive, and you feel a growing sense of unease.")
	room2 := NewRoom(2, "Room 0", "It's dark and damp, and you can't see anything. You hear a faint dripping sound.\n\nYou feel a sense of dread, and you're not sure why.")

	room1.AddExit("down", room2)
	room2.AddExit("up", room1)

	game.rooms[room1.ID] = room1
	game.rooms[room2.ID] = room2

	go game.loop()

	return game
}

func (g *Game) RemovePlayer(player Player) {
	g.removePlayerChan <- player
	room, ok := g.findPlayerRoom(player)
	if !ok {
		room.RemovePlayer(player)
	}
}

var lastTime time.Time

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
		case player := <-g.addPlayerChan:
			g.clients[player.Client.RemoteAddr()] = player
		case player := <-g.removePlayerChan:
			delete(g.clients, player.RemoteAddr())
		case playerCmd := <-g.commandChan:
			g.ExecuteCommand(playerCmd.player, playerCmd.command)
		default:
			deltaTime := calculateDeltaTime()
			for _, system := range g.systems {
				system.Update(g.entities, deltaTime)
			}
		}
	}
}

func (g *Game) findPlayerRoom(player Player) (*Room, bool) {
	for _, room := range g.rooms {
		if _, ok := room.Players[player.RemoteAddr()]; ok {
			return room, true
		}
	}
	return nil, false
}

func (g *Game) handleExit(player Player) {
	g.RemovePlayer(player)
	player.CloseConnection()
	departureMessage := fmt.Sprintf("%s has left the server.", player.Name())
	for _, p := range g.clients {
		p.SendMessage(departureMessage)
	}
}

func (g *Game) handleLook(player Player) {
	room, ok := g.findPlayerRoom(player)
	if !ok {
		message := room.Description
	} else {
		message := "You aren't sure where you are."
	}
	player.SendMessage(message)
}

func (g *Game) handleMove(player Player, direction string) {
	room, ok := g.findPlayerRoom(player)
	if !ok {
		player.SendMessage("Error: Cannot find your current room.")
		return
	}

	if room.Exits[direction] == nil {
		player.SendMessage("You can't go that way.")
		return
	}

	departureMessage := fmt.Sprintf("%s leaves %s.", player.Name(), direction)
	for _, p := range room.Players {
		if p.Name() != player.Name() {
			p.SendMessage(departureMessage)
		}
	}

	g.movePlayerToRoom(player, room.Exits[direction])
}

func (g *Game) handleSay(player Player, cmd *Command) {
	if len(cmd.Arguments) == 0 {
		player.SendMessage("Say what?")
		return
	}

	message := strings.Join(cmd.Arguments, " ")
	room, ok := g.findPlayerRoom(player)
	if !ok {
		player.SendMessage("Error: Cannot find your current room.")
		return
	}

	broadcastMessage := fmt.Sprintf("%s says: %s", player.Name(), message)
	for _, p := range room.Players {
		if p.Name() != player.Name() {
			p.SendMessage(broadcastMessage)
		}
	}

	player.SendMessage(fmt.Sprintf("You say: %s", message))
}

func (g *Game) handleScan(player Player) {
	room, ok := g.findPlayerRoom(player)
	if !ok {
		player.SendMessage("Error: Cannot find your current room.")
		return
	}

	if len(room.Exits) == 0 {
		player.SendMessage("There are no exits in this room.")
	} else {
		exits := make([]string, 0, len(room.Exits))
		for direction := range room.Exits {
			exits = append(exits, direction)
		}

		log.Printf("Room: %s (%d)", room.Name, room.ID)

		player.SendMessage(fmt.Sprintf("Exits: %s", strings.Join(exits, ", ")))
	}
}

func (g *Game) handleShout(player Player, cmd *Command) {
	message := strings.Join(cmd.Arguments, " ")
	broadcastMessage := fmt.Sprintf("%s shouts: %s", player.Name(), message)
	for _, p := range g.clients {
		p.SendMessage(broadcastMessage)
	}
}

func (g *Game) movePlayerToRoom(player Player, room *Room) {
	currentRoom, ok := g.findPlayerRoom(player)
	if !ok {
		log.Printf("Player %s is not in any room??", player.RemoteAddr())
	}
	currentRoom.RemovePlayer(player)

	enterMessage := fmt.Sprintf("%s has entered the room.", player.Name())
	for _, p := range currentRoom.Players {
		p.SendMessage(enterMessage)
	}

	room.AddPlayer(player)

	player.SendMessage(room.Description)
}
