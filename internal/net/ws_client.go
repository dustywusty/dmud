package net

import (
	"net/http"
	"net/url"
	"strings"

	"dmud/internal/common"
	"dmud/internal/game"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		parsedOrigin, err := url.Parse(origin)
		if err != nil {
			log.Error().Err(err).Msg("Error parsing Origin header")
			return false
		}

		if strings.ToLower(parsedOrigin.Hostname()) != "localhost" {
			log.Error().Msg("Origin is not localhost")
			return false
		}

		return true
	},
}

type WSClient struct {
	conn *websocket.Conn
	game *game.Game
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