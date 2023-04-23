package game

import (
  "strings"
  "fmt"
)

type Game struct {
  clients          map[string]Player
  addPlayerChan    chan Player
  removePlayerChan chan Player
  commandChan      chan *PlayerCommand
}

type PlayerCommand struct {
  player   Player
  command *Command
}

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

func (g *Game) AddPlayer(player Player) {
  g.addPlayerChan <- player
}

func (g *Game) RemovePlayer(player Player) {
  g.removePlayerChan <- player
}

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

func (g *Game) handleShout(player Player, cmd *Command) {
  message := strings.Join(cmd.Arguments, " ")
  broadcastMessage := fmt.Sprintf("%s shouts: %s\n", player.Name(), message)
  for _, p := range g.clients {
    p.SendMessage(broadcastMessage)
  }
}

func (g *Game) handleLook(player Player) {
  message := "The room is white and nondescript, with claw marks on the ceiling that you can't help but notice. In one corner, there's a pile of Ayn Rand books.\n\nThe atmosphere is oppressive, and you feel a growing sense of unease."
  player.SendMessage(message)
}
