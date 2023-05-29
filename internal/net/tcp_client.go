package net

import (
	"bufio"
	"errors"
	"io"
	"log"
	"net"
	"strings"

	"dmud/internal/common"
	"dmud/internal/ecs"
	"dmud/internal/game"
)

type TCPClient struct {
	conn     net.Conn
	game     *game.Game
	id       string
	playerId ecs.EntityID
	reader   *bufio.Reader
}

var _ common.Client = (*TCPClient)(nil)

// /////////////////////////////////////////////////////////////////////////////////////////////
// Public
//

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

func (client *TCPClient) GetMessage(maxLength int) (string, error) {
	msg, err := client.reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return "", errors.New("Client disconnected")
		}
		return "", err
	}

	msg = strings.TrimSpace(msg)

	if len(msg) > maxLength {
		return "", errors.New("message exceeds maximum length")
	}

	return msg, nil
}

func (c *TCPClient) ID() string {
	return c.id
}

func (c *TCPClient) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}

func (c *TCPClient) SendMessage(msg string) {
	_, err := c.conn.Write([]byte("\b\b" + msg + "\n\n> "))
	if err != nil {
		log.Printf("Error sending message to TCPClient: %v", err)
		return
	}
}

// /////////////////////////////////////////////////////////////////////////////////////////////
// Private
//

func (c *TCPClient) handleRequest() {
	g := c.game
	r := bufio.NewReader(c.conn)
	for {
		message, err := r.ReadString('\n')
		if err != nil {
			c.conn.Close()
			g.HandleDisconnect(c)
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

		g.CommandChan <- game.ClientCommand{Command: command, Client: c}
	}
}
