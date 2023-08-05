package net

import (
	"bufio"
	"net"
	"strings"

	"dmud/internal/common"
	"dmud/internal/ecs"
	"dmud/internal/game"

	"github.com/rs/zerolog/log"
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
		log.Error().Err(err).Msg("Error closing connection")
		return err
	}
	log.Printf("Closed connection to %s", c.RemoteAddr())
	return nil
}

func (c *TCPClient) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}

func (c *TCPClient) SendMessage(msg string) {
	_, err := c.conn.Write([]byte("\b\b" + msg + "\n\n> "))
	if err != nil {
		log.Error().Err(err).Msg("Error sending message to TCPClient: %v")
	}
}

func (c *TCPClient) HandleRequest() {
	g := c.game
	r := bufio.NewReader(c.conn)
	for {
		message, err := r.ReadString('\n')
		if err != nil {
			c.conn.Close()
			g.HandleDisconnect(c)
			return
		}

		log.Trace().Msgf("Received message from %s: %s", c.RemoteAddr(), message)

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
