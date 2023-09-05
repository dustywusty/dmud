package net

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"

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
			log.Info().Msgf("Origin %s is not localhost", parsedOrigin.Hostname())
			return false
		}

		return true
	},
}

// -----------------------------------------------------------------------------

type WSClient struct {
	status common.ConnectionStatus
	conn   *websocket.Conn
	game   *game.Game
	mu     sync.Mutex
}

var _ common.Client = (*WSClient)(nil)

// -----------------------------------------------------------------------------

func (c *WSClient) CloseConnection() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.status = common.Disconnecting

	log.Info().Msgf("Trying to close connection to %s", c.RemoteAddr())

	if c.conn.UnderlyingConn() == nil {
		log.Info().Msgf("Connection to %s already closed", c.RemoteAddr())
		return nil
	}
	err := c.conn.Close()
	if err != nil {
		log.Error().Err(err).Msg("Error closing connection")
		return err
	}

	c.status = common.Disconnected

	log.Info().Msgf("Closed connection to %s", c.RemoteAddr())
	return nil
}

// -----------------------------------------------------------------------------

func (c *WSClient) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}

// -----------------------------------------------------------------------------

func (c *WSClient) SendMessage(msg string) {
	err := c.conn.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		log.Error().Msgf("Error sending message %s to %s: %s", msg, c.RemoteAddr(), err)
	} else {
		log.Trace().Msgf("Sent message to %s: %s", c.RemoteAddr(), msg)
	}
}

// -----------------------------------------------------------------------------

func (c *WSClient) HandleRequest() {
	g := c.game
	for {
		messageType, p, err := c.conn.ReadMessage()
		if err != nil {
			c.mu.Lock()
			if c.status == common.Disconnected {
				c.mu.Unlock()
				return
			}
			c.mu.Unlock()
			log.Error().Err(err).Msgf("Error reading message from %s", c.RemoteAddr())
			g.RemovePlayerChan <- c
			return
		}
		// Check for slurs on all player input
		// For now just disconnect them immediately
		inputLower := strings.ToLower(strings.TrimSpace(string(p)))
		for _, slur := range util.Slurs {
			re, err := regexp.Compile(`\b` + regexp.QuoteMeta(slur) + `\b`)
			if err != nil {
				log.Error().Msgf("Error compiling regex: %s", err)
				continue
			}
			// If it appears as a word in the input, disconnect the player
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
