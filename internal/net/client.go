package net

import (
	"bufio"
	"dmud/internal/game"
	"log"
	"net"
	"strings"
)

type Client struct {
	conn   net.Conn
	game   *game.Game
	player *game.Player
}

func (client *Client) RemoteAddr() string {
	return client.conn.RemoteAddr().String()
}

func (client *Client) SendMessage(msg string) {
	client.conn.Write([]byte("\b\b" + msg + "\n\n> "))
}

func (client *Client) CloseConnection() {
	client.conn.Close()
}

func (client *Client) handleRequest() {
	reader := bufio.NewReader(client.conn)

	for {
		message, err := reader.ReadString('\n')

		if err != nil {
			client.conn.Close()
			client.game.RemovePlayer(client.player)
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

func parseCommand(message string) *game.Command {
	words := strings.Fields(message)
	if len(words) == 0 {
		return nil
	}

	cmd := &game.Command{
		Name:      strings.ToLower(words[0]),
		Arguments: words[1:],
	}

	return cmd
}
