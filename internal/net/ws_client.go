package net

import (
	"errors"
	"strings"

	"dmud/internal/common"
	"dmud/internal/ecs"
	"dmud/internal/game"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{} // use default options

type WSClient struct {
	conn     *websocket.Conn
	game     *game.Game
	id       string
	playerId ecs.EntityID
}

var _ common.Client = (*WSClient)(nil)

func (c *WSClient) CloseConnection() error {
	msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "Goodbye!")
	if err := c.conn.WriteMessage(websocket.CloseMessage, msg); err != nil {
		log.Error().Err(err).Msg("Error closing connection")
		return err
	}
	log.Printf("Closed connection to %s", c.RemoteAddr())
	return nil
}

func (c *WSClient) GetMessage(maxLength int) (string, error) {
	messageType, p, err := c.conn.ReadMessage()
	if err != nil {
		log.Error().Err(err).Msg("Error reading message")
		return "", err
	}

	if messageType == websocket.TextMessage {
		msg := strings.TrimSpace(string(p))

		if len(msg) > maxLength {
			return "", errors.New("message exceeds maximum length")
		}

		return msg, nil
	}

	return "", nil
}

func (c *WSClient) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}

func (c *WSClient) SendMessage(msg string) {
	err := c.conn.WriteMessage(websocket.TextMessage, []byte("\b\b"+msg+"\n\n> "))
	if err != nil {
		log.Error().Err(err).Msg("Error sending message to WSClient")
	}
}

func (c *WSClient) HandleRequest() {
	g := c.game
	for {
		messageType, p, err := c.conn.ReadMessage()
		if err != nil {
			c.CloseConnection()
			g.HandleDisconnect(c)
			return
		}

		if messageType == websocket.TextMessage {
			log.Trace().Msgf("Received message from %s: %s", c.RemoteAddr(), p)

			parts := strings.SplitN(strings.TrimSpace(string(p)), " ", 2)
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
}
