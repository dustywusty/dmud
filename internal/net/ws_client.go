package net

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"dmud/internal/common"
	"dmud/internal/game"
	"dmud/internal/util"

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
	err := c.conn.Close()
	if err != nil {
		log.Error().Err(err).Msg("Error closing connection")
		return err
	}

	log.Info().Msgf("Closed connection to %s", c.RemoteAddr())
	return nil
}

func (c *WSClient) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}

func (c *WSClient) SendMessage(msg string) {
	log.Trace().Msgf("Sending message to %s: %s", c.RemoteAddr(), msg)

	err := c.conn.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		log.Error().Msgf("Error sending message %s to %s: %s", msg, c.RemoteAddr(), err)
	} else {
		log.Trace().Msgf("Sent message to %s: %s", c.RemoteAddr(), msg)
	}
}

func (c *WSClient) HandleRequest() {
	g := c.game
	for {
		messageType, p, err := c.conn.ReadMessage()
		if err != nil {
			g.RemovePlayerChan <- c
			return
		}

		inputLower := strings.ToLower(strings.TrimSpace(string(p)))
		for _, slur := range util.Slurs {
			re, err := regexp.Compile(`\b` + regexp.QuoteMeta(slur) + `\b`)
			if err != nil {
				log.Error().Msgf("Error compiling regex: %s", err)
				continue
			}
			if re.MatchString(inputLower) {
				log.Warn().Msgf("Slur detected in message from %s. Message rejected.", c.RemoteAddr())
				g.RemovePlayerChan <- c
				return
			}
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
