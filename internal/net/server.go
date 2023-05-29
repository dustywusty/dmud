package net

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"sync"

	"dmud/internal/common"
	"dmud/internal/game"
	"dmud/internal/util"
)

type ServerConfig struct {
	Host string
	Port string
}

type Server struct {
	host        string
	port        string
	connections map[string]common.Client
	game        *game.Game
	mu          sync.Mutex
}

func NewServer(config *ServerConfig) *Server {
	return &Server{
		host:        config.Host,
		port:        config.Port,
		connections: make(map[string]common.Client),
	}
}

///////////////////////////////////////////////////////////////////////////////////////////////
// Public
//

func (s *Server) Run() {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", s.host, s.port))
	if err != nil {
		log.Fatal(err)
	}
	defer func(listener net.Listener) {
		err := listener.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(listener)

	log.Printf("Listening on %s:%s", s.host, s.port)

	s.game = game.NewGame()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		remoteAddr := conn.RemoteAddr().String()
		log.Printf("Accepted connection from %s", remoteAddr)

		client := &TCPClient{
			conn:   conn,
			game:   s.game,
			reader: bufio.NewReader(conn),
		}

		s.mu.Lock()
		s.connections[remoteAddr] = client
		s.mu.Unlock()

		s.handleLogin(client)

		s.game.AddPlayer(client)

		go client.handleRequest()
	}
}

func (s *Server) handleLogin(c common.Client) {
	var name, password string
	for {
		c.SendMessage(util.WelcomeBanner)
		c.SendMessage("What do we call you?")

		name, err := c.GetMessage(32)
		if err != nil {
			log.Fatal(err)
		}

		if len(name) > 32 || !util.IsAlphaNumeric(name) {
			continue
		}
		break
	}

	for {
		c.SendMessage("What is your password?")
		password, err := c.GetMessage(256)
		if err != nil {
			log.Fatal(err)
		}

		if len(password) <= 0 || len(password) > 256 {
			continue
		}
		break
	}

	log.Printf("name: %s, password: %s", name, password)
}
