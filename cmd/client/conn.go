package main

import (
	"bufio"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gorilla/websocket"
)

type transport int

const (
	transportWS transport = iota
	transportTCP
)

func (t transport) String() string {
	if t == transportTCP {
		return "TCP"
	}
	return "WebSocket"
}

// resolveAddr fills in a sensible default host:port for the chosen transport
// when the user did not pass -addr.
func resolveAddr(addr string, t transport) string {
	if addr != "" {
		return addr
	}
	if t == transportTCP {
		return "localhost:3333"
	}
	return "localhost:8080"
}

// --- Bubble Tea messages emitted by the connection ---

// connectedMsg is delivered once the dial succeeds.
type connectedMsg struct{}

// disconnectedMsg is delivered when the connection closes or fails to open.
type disconnectedMsg struct{ err error }

// chunkMsg carries one logical server message. Over WebSocket this is a single
// frame (which may contain several lines, e.g. a room description); over TCP it
// is a single line.
type chunkMsg struct{ raw string }

// reconnectMsg fires after the auto-reconnect backoff to retry the connection.
type reconnectMsg struct{}

// conn abstracts the WebSocket and TCP transports behind a channel of Bubble
// Tea messages plus a Send method for outbound commands.
type conn struct {
	transport transport
	addr      string
	events    chan tea.Msg

	mu     sync.Mutex
	ws     *websocket.Conn
	tcp    net.Conn
	closed bool
}

func newConn(t transport, addr string) *conn {
	return &conn{
		transport: t,
		addr:      addr,
		events:    make(chan tea.Msg, 64),
	}
}

// dial opens the connection and starts the background reader. It is safe to
// call from a Bubble Tea command.
func (c *conn) dial() error {
	if c.transport == transportTCP {
		return c.dialTCP()
	}
	return c.dialWS()
}

func (c *conn) dialWS() error {
	raw := c.addr
	if !strings.Contains(raw, "://") {
		raw = "ws://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if u.Path == "" || u.Path == "/" {
		u.Path = "/ws"
	}

	dialer := *websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second
	ws, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.ws = ws
	c.mu.Unlock()

	go c.readWS()
	return nil
}

func (c *conn) readWS() {
	for {
		_, data, err := c.ws.ReadMessage()
		if err != nil {
			c.emit(disconnectedMsg{err: err})
			c.markClosed()
			return
		}
		text := strings.TrimRight(string(data), "\r\n")
		if text == "" {
			continue
		}
		c.emit(chunkMsg{raw: text})
	}
}

func (c *conn) dialTCP() error {
	tcp, err := net.DialTimeout("tcp", c.addr, 10*time.Second)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.tcp = tcp
	c.mu.Unlock()

	go c.readTCP()
	return nil
}

func (c *conn) readTCP() {
	r := bufio.NewReader(c.tcp)
	for {
		line, err := r.ReadString('\n')
		// The TCP server prints a "> " prompt with no newline; surface whatever
		// arrived before reporting the error so partial lines are not lost.
		if line != "" {
			text := strings.TrimRight(line, "\r\n")
			text = strings.TrimPrefix(text, "> ")
			if strings.TrimSpace(text) != "" {
				c.emit(chunkMsg{raw: text})
			}
		}
		if err != nil {
			c.emit(disconnectedMsg{err: err})
			c.markClosed()
			return
		}
	}
}

// Send writes one command line to the server.
func (c *conn) Send(line string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return net.ErrClosed
	}
	if c.ws != nil {
		return c.ws.WriteMessage(websocket.TextMessage, []byte(line))
	}
	if c.tcp != nil {
		_, err := c.tcp.Write([]byte(line + "\n"))
		return err
	}
	return net.ErrClosed
}

func (c *conn) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	if c.ws != nil {
		_ = c.ws.Close()
	}
	if c.tcp != nil {
		_ = c.tcp.Close()
	}
}

func (c *conn) markClosed() {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
}

// emit pushes an event to the model unless the connection is already closed.
func (c *conn) emit(msg tea.Msg) {
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed {
		return
	}
	c.events <- msg
}

// --- Bubble Tea commands that drive the connection ---

// connectCmd dials the server and reports success or failure.
func connectCmd(c *conn) tea.Cmd {
	return func() tea.Msg {
		if err := c.dial(); err != nil {
			return disconnectedMsg{err: err}
		}
		return connectedMsg{}
	}
}

// waitForEvent blocks on the next connection event so the model can react to
// it. The model re-issues this command after handling each event to keep the
// stream flowing.
func waitForEvent(c *conn) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-c.events
		if !ok {
			return disconnectedMsg{}
		}
		return msg
	}
}

// sendCmd sends a command line off the UI goroutine.
func sendCmd(c *conn, line string) tea.Cmd {
	return func() tea.Msg {
		_ = c.Send(line)
		return nil
	}
}
