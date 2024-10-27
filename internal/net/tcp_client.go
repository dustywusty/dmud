package net

import (
	"bufio"
	"net"
	"strings"

	"dmud/internal/common"
	"dmud/internal/game"

	"github.com/rs/zerolog/log"
)

type TCPClient struct {
	conn net.Conn
	game *game.Game
}

var _ common.Client = (*TCPClient)(nil)

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

func (c *TCPClient) HandleRequest() {
	g := c.game
	r := bufio.NewReader(c.conn)

	for {
		message, err := r.ReadString('\n')
		if err != nil {
			log.Error().Err(err).Msg("Error reading string from TCPClient")
			g.HandleDisconnect(c)
			return
		}

		message = strings.TrimSpace(message)

		log.Trace().Msgf("Received message from %s: %s", c.RemoteAddr(), message)

		parts := strings.SplitN(message, " ", 2)
		cmd := parts[0]
		var args []string
		if len(parts) > 1 {
			args = strings.Split(parts[1], " ")
		}

		c.SendMessage("\n")

		g.ExecuteCommandChan <- game.ClientCommand{
			Client: c,
			Cmd:    cmd,
			Args:   args,
		}
	}
}

func (c *TCPClient) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}

func (c *TCPClient) SendMessage(msg string) {
	_, err := c.conn.Write([]byte(msg))
	if err != nil {
		log.Error().Err(err).Msg("Error sending message to TCPClient: %v")
	}
}
