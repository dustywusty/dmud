package net

import (
	"bufio"
	"log"
	"net"
	"strings"

	"dmud/internal/common"
	"dmud/internal/ecs"
	"dmud/internal/game"
)

type Client struct {
	conn     net.Conn   // the network connection to the client
	game     *game.Game // reference to the game object
	id       string     // unique identifier for the client
	playerId ecs.EntityID
}

// Check that Client implements the common.Client interface.
var _ common.Client = (*Client)(nil)

// ID returns the client's unique identifier.
func (c *Client) ID() string {
	return c.id
}

// RemoteAddr returns the remote network address of the client.
func (c *Client) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}

// SendMessage sends a message to the client.
func (c *Client) SendMessage(msg string) {
	_, err := c.conn.Write([]byte("\b\b" + msg + "\n\n> "))
	if err != nil {
		log.Printf("Error sending message to client: %v", err)
		return
	}
}

// CloseConnection closes the network connection to the client.
func (c *Client) CloseConnection() error {
	c.conn.Write([]byte("\nGoodbye!\n\n"))
	err := c.conn.Close()
	if err != nil {
		log.Printf("Error closing connection: %v", err)
		return err
	}
	log.Printf("Closed connection to %s", c.RemoteAddr())
	return nil
}

// handleRequest reads and handles requests from the client.
func (c *Client) handleRequest() {
	reader := bufio.NewReader(c.conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			c.conn.Close()
			c.game.RemovePlayer(c)
			return
		}

		parts := strings.SplitN(strings.TrimSpace(message), " ", 2)
		cmd := parts[0]
		var args []string
		if len(parts) > 1 {
			args = strings.Split(parts[1], " ")
		}

		command := &game.Command{
			Cmd:    cmd,
			Args:   args,
			Client: c,
		}

		c.game.CommandChan <- command

		c.conn.Write([]byte("\n> "))
	}
}
