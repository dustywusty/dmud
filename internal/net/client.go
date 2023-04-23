package net

import (
  "bufio"
  "dmud/internal/game"
  "dmud/internal/util"
  "log"
  "math/rand"
  "net"
  "strings"
  "time"
)

// Client represents a connected player in the MUD server.
type Client struct {
  conn net.Conn
  game *game.Game
  name string
}

// Name returns the client's name.
func (client *Client) Name() string {
  return client.name
}

// RemoteAddr returns the client's remote address.
func (client *Client) RemoteAddr() string {
  return client.conn.RemoteAddr().String()
}

// SendMessage sends a message to the client.
func (client *Client) SendMessage(msg string) {
	client.conn.Write([]byte("\b\b" + msg + "\n\n> "))
}

func (client *Client) CloseConnection() {
  client.conn.Close()
}

// generateRandomName generates a random name for the client.
func (client *Client) generateRandomName() string {
  rand.Seed(time.Now().UnixNano())

  noun := util.Nouns[rand.Intn(len(util.Nouns))]
  verb1 := util.AdjectiveVerbs1[rand.Intn(len(util.AdjectiveVerbs1))]
  verb2 := util.AdjectiveVerbs2[rand.Intn(len(util.AdjectiveVerbs2))]

  return verb1 + "-" + verb2 + "-" + noun
}

// handleRequest processes incoming messages from the client and executes game commands.
func (client *Client) handleRequest() {
  reader := bufio.NewReader(client.conn)

  for {
    message, err := reader.ReadString('\n')

    if err != nil {
      client.conn.Close()
      client.game.RemovePlayer(client)
      return
    }

    cmd := parseCommand(message)
    if cmd != nil {
      log.Printf("Received command: %s, args: %s", cmd.Name, cmd.Arguments)
      client.game.ExecuteCommand(client, cmd)
    } else {
      log.Printf("Invalid command: %s", message)
    }
  }
}

// parseCommand converts an incoming message string into a game.Command object.
func parseCommand(message string) *game.Command {
  words := strings.Fields(message)
  if len(words) == 0 {
    return nil
  }

  cmd := &game.Command{
    Name: strings.ToLower(words[0]),
    Arguments: words[1:],
  }

  return cmd
}
