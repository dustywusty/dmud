package net

import (
	"bufio"
	"log"
	"net"
	"strings"

	"dmud/internal/common"
	"dmud/internal/game"
)

type Client struct {
	conn net.Conn
	game *game.Game
	id   string
}

var _ common.Client = (*Client)(nil)

func (c *Client) ID() string {
	return c.id
}

func (c *Client) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}

func (c *Client) SendMessage(msg string) {
	_, err := c.conn.Write([]byte("\b\b" + msg + "\n\n> "))
	if err != nil {
		log.Printf("Error sending message to client: %v", err)
		return
	}
}

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
