package game

import (
  "strings"
  "fmt"
)

// Game represents the main game state, managing connected clients and processing player commands.
type Game struct {
  clients          map[string]Player
  addPlayerChan    chan Player
  removePlayerChan chan Player
  commandChan      chan *PlayerCommand
}

// PlayerCommand is a structure that holds a player and their associated command.
type PlayerCommand struct {
  player   Player
  command *Command
}

// NewGame initializes a new game instance and starts its loop in a separate goroutine.
func NewGame() *Game {
  game := &Game{
    clients:          make(map[string]Player),
    addPlayerChan:    make(chan Player),
    removePlayerChan: make(chan Player),
    commandChan:      make(chan *PlayerCommand),
  }

  go game.loop()

  return game
}

// loop is the main event loop for the game, handling player additions, removals, and command execution.
func (g *Game) loop() {
  for {
    select {
    case player := <-g.addPlayerChan:
      g.clients[player.RemoteAddr()] = player
    case player := <-g.removePlayerChan:
      delete(g.clients, player.RemoteAddr())
    case playerCmd := <-g.commandChan:
      g.ExecuteCommand(playerCmd.player, playerCmd.command)
    }
  }
}

// AddPlayer adds a new player to the game.
func (g *Game) AddPlayer(player Player) {
  g.addPlayerChan <- player
}

// RemovePlayer removes a player from the game.
func (g *Game) RemovePlayer(player Player) {
  g.removePlayerChan <- player
}

// ExecuteCommand processes a player's command and performs the corresponding action in the game.
func (g *Game) ExecuteCommand(player Player, cmd *Command) {
  switch cmd.Name {
  case "shout":
    g.handleShout(player, cmd)
  case "look", "scan", "exits":
    g.handleLook(player)
  default:
    player.SendMessage("Unknown command.")
  }
}

// handleShout broadcasts a message from the player to all connected clients.
func (g *Game) handleShout(player Player, cmd *Command) {
  message := strings.Join(cmd.Arguments, " ")
  broadcastMessage := fmt.Sprintf("%s shouts: %s\n", player.Name(), message)
  for _, p := range g.clients {
    p.SendMessage(broadcastMessage)
  }
}

// handleLook sends a description of the current room to the player.
func (g *Game) handleLook(player Player) {
  message := "The room is white and nondescript, with claw marks on the ceiling that you can't help but notice. In one corner, there's a pile of Ayn Rand books.\n\nThe atmosphere is oppressive, and you feel a growing sense of unease."
  player.SendMessage(message)
}
