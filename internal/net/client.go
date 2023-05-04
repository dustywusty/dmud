package net

import (
	"bufio"
	"dmud/internal/game"
	"log"
	"net"
	"strings"
)

type Client struct {
	conn net.Conn
	game *game.Game
}

func (c *Client) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}

func (c *Client) SendMessage(msg string) {
	c.conn.Write([]byte("\b\b" + msg + "\n\n> "))
}

func (c *Client) CloseConnection() {
	c.conn.Close()
}

func (c *Client) handleRequest() {
	reader := bufio.NewReader(c.conn)

	for {
		message, err := reader.ReadString('\n')

		if err != nil {
			c.conn.Close()
			c.game.RemovePlayer(c)
			return
		}

		cmd := parseCommand(message)
		if cmd != nil {
			log.Printf("Received command: %s, args: %s", cmd.Name, cmd.Arguments)
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
