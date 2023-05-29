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

type TCPClient struct {
	conn     net.Conn   // the network connection to the TCPClient
	game     *game.Game // reference to the game object
	id       string     // unique identifier for the TCPClient
	playerId ecs.EntityID
}

// Check that TCPClient implements the common.TCPClient interface.
var _ common.Client = (*TCPClient)(nil)

// ID returns the TCPClient's unique identifier.
func (c *TCPClient) ID() string {
	return c.id
}

// RemoteAddr returns the remote network address of the TCPClient.
func (c *TCPClient) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}

// SendMessage sends a message to the TCPClient.
func (c *TCPClient) SendMessage(msg string) {
	_, err := c.conn.Write([]byte("\b\b" + msg + "\n\n> "))
	if err != nil {
		log.Printf("Error sending message to TCPClient: %v", err)
		return
	}
}

// CloseConnection closes the network connection to the TCPClient.
func (c *TCPClient) CloseConnection() error {
	c.conn.Write([]byte("\nGoodbye!\n\n"))
	err := c.conn.Close()
	if err != nil {
		log.Printf("Error closing connection: %v", err)
		return err
	}
	log.Printf("Closed connection to %s", c.RemoteAddr())
	return nil
}

// handleRequest reads and handles requests from the TCPClient.
func (c *TCPClient) handleRequest() {
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

		command := game.Command{
			Cmd:  cmd,
			Args: args,
		}

		c.game.CommandChan <- game.ClientCommand{Command: command, Client: c}
	}
}
