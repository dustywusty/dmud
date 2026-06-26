// Command client is a terminal MUD client for the dmud server.
//
// It speaks the same wire protocol as the server's WebSocket/TCP clients and
// arranges the stream into adjustable panels: a main output view, a chat /
// communication view, a room + exits view, and a status bar driven by the
// server's STATE| frames.
//
// Usage:
//
//	dmud-client                       # WebSocket to ws://localhost:8080/ws
//	dmud-client -addr host:8080       # WebSocket to a remote host
//	dmud-client -tcp                  # raw TCP to localhost:3333
//	dmud-client -tcp -addr host:3333  # raw TCP to a remote host
//	dmud-client -login <uuid>         # restore a saved character on connect
//	dmud-client -name Gandalf         # set your name on connect
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	addr := flag.String("addr", "", "server address host:port (defaults to localhost on the transport's default port)")
	useTCP := flag.Bool("tcp", false, "use raw TCP (port 3333) instead of WebSocket (port 8080)")
	name := flag.String("name", "", "send `name <value>` immediately after connecting")
	login := flag.String("login", "", "send `login <uuid>` immediately after connecting to restore a character")
	flag.Parse()

	tr := transportWS
	if *useTCP {
		tr = transportTCP
	}
	target := resolveAddr(*addr, tr)
	c := newConn(tr, target)

	// Resume a saved character: use -login if given, otherwise the id the server
	// handed us last time (stored per server in identity.json).
	loginID := *login
	if loginID == "" {
		loginID = loadIdentity(target)
	}

	m := newModel(c, autoCommands(loginID, *name))
	m.applyConfig(loadConfig())
	m.identity = loginID
	m.mapper.restore(loadMapData(mapFilePath(c.addr)))
	m.history = loadHistory()
	m.histPos = len(m.history)

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "dmud-client:", err)
		os.Exit(1)
	}
}

// autoCommands returns the commands to send right after the connection opens.
// A login is sent before a name so a restored character keeps its saved name.
func autoCommands(login, name string) []string {
	var cmds []string
	if login != "" {
		cmds = append(cmds, "login "+login)
	}
	if name != "" {
		cmds = append(cmds, "name "+name)
	}
	return cmds
}
