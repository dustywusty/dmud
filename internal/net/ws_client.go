package net

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"dmud/internal/common"
	"dmud/internal/game"
	"dmud/internal/util"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		o := r.Header.Get("Origin")
		if o == "" {
			// Non-browser clients often omit Origin; allow.
			return true
		}
		u, err := url.Parse(o)
		if err != nil {
			log.Error().Err(err).Str("origin", o).Msg("bad Origin")
			return false
		}
		oh := strings.ToLower(u.Host)
		rh := strings.ToLower(r.Host) // includes host[:port]
		// strip port from r.Host for comparison
		if i := strings.IndexByte(rh, ':'); i >= 0 {
			rh = rh[:i]
		}

		// Same-origin?
		if oh == rh {
			return true
		}
		// Cloud Run default domains
		if strings.HasSuffix(oh, ".run.app") || strings.HasSuffix(oh, ".a.run.app") {
			return true
		}
		// Your GH Pages front-end (keep this)
		if oh == "dusty.wtf" {
			return true
		}
		if oh == "dusty-wtf.pages.dev" || strings.HasSuffix(oh, ".dusty-wtf.pages.dev") {
			return true
		}
		log.Info().Msgf("Rejected Origin %s for host %s", oh, rh)
		return false
	},
}

type WSClient struct {
	status     common.ConnectionStatus
	conn       *websocket.Conn
	game       *game.Game
	mu         sync.Mutex
	writeMu    sync.Mutex // serialize writes; gorilla allows only one writer
	realIP     string     // actual client IP (from proxy headers if behind proxy)
}

func (c *WSClient) SupportsPrompt() bool { return false }

var _ common.Client = (*WSClient)(nil)

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

const (
	pongWait   = 60 * time.Second
	pingPeriod = 30 * time.Second // must be < pongWait
	writeWait  = 10 * time.Second
)

func (c *WSClient) HandleRequest() {
	g := c.game
	slurRegexes := compileSlurRegexes()

	// --- keep the read side alive & detect dead peers
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	// --- ping loop (stops when read loop returns)
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.writeMu.Lock()
				_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					c.writeMu.Unlock()
					// triggers removal on read side shortly
					return
				}
				c.writeMu.Unlock()
			case <-done:
				return
			}
		}
	}()

	for {
		messageType, p, err := c.readMessage()
		if err != nil {
			close(done)
			handleReadError(err, c)
			return
		}

		// drop empty/whitespace-only frames (prevents “blank command” noise)
		if len(strings.TrimSpace(string(p))) == 0 {
			continue
		}

		if containsSlur(slurRegexes, p) {
			log.Warn().Msgf("Slur detected in message from %s. Message rejected.", c.RemoteAddr())
			close(done)
			g.RemovePlayerChan <- c
			return
		}

		if messageType == websocket.TextMessage {
			processTextMessage(p, c)
		} else {
			log.Warn().Msgf("Unknown message type %d from %s", messageType, c.RemoteAddr())
		}
	}
}

func (c *WSClient) RemoteAddr() string {
	// Return real IP if we extracted it from proxy headers, otherwise fallback to connection address
	if c.realIP != "" {
		return c.realIP
	}
	return c.conn.RemoteAddr().String()
}

func (c *WSClient) SendMessage(msg string) {
	s := msg

	c.writeMu.Lock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	err := c.conn.WriteMessage(websocket.TextMessage, []byte(s))
	c.writeMu.Unlock()

	if err != nil {
		log.Error().Msgf("Error sending message to %s: %v", c.RemoteAddr(), err)
	} else {
		log.Trace().Msgf("Sent message to %s", c.RemoteAddr())
	}
}

func containsSlur(slurRegexes []*regexp.Regexp, message []byte) bool {
	inputLower := strings.ToLower(strings.TrimSpace(string(message)))
	for _, re := range slurRegexes {
		if re.MatchString(inputLower) {
			return true
		}
	}
	return false
}

func compileSlurRegexes() []*regexp.Regexp {
	var slurRegexes []*regexp.Regexp
	for _, slur := range util.Slurs {
		re, err := regexp.Compile(`\b` + regexp.QuoteMeta(slur) + `\b`)
		if err != nil {
			log.Error().Msgf("Error compiling regex: %s", err)
			continue
		}
		slurRegexes = append(slurRegexes, re)
	}
	return slurRegexes
}

func handleReadError(err error, c *WSClient) {
	g := c.game
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.status == common.Disconnected {
		return
	}
	c.status = common.Disconnected
	log.Error().Err(err).Msgf("Error reading message from %s", c.RemoteAddr())
	g.RemovePlayerChan <- c
}

func processTextMessage(p []byte, c *WSClient) {
	log.Trace().Msgf("Received message from %s: %s", c.RemoteAddr(), p)

	g := c.game

	parts := strings.SplitN(strings.TrimSpace(string(p)), " ", 2)
	cmd := parts[0]

	var args []string
	if len(parts) > 1 {
		args = strings.Split(parts[1], " ")
	}

	clientCommand := game.ClientCommand{
		Cmd:    cmd,
		Args:   args,
		Client: c,
	}

	g.ExecuteCommandChan <- clientCommand
}

func (c *WSClient) readMessage() (messageType int, p []byte, err error) {
	messageType, p, err = c.conn.ReadMessage()
	return
}
