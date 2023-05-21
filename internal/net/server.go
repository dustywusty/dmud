package net

import (
	"fmt"
	"log"
	"net"
	"sync"

	"dmud/internal/game"
)

type ServerConfig struct {
	Host string
	Port string
}

func NewServer(config *ServerConfig) *Server {
	return &Server{
		host:        config.Host,
		port:        config.Port,
		connections: make(map[string]*Client),
	}
}

type Server struct {
	host        string
	port        string
	connections map[string]*Client
	game        *game.Game
	mu          sync.Mutex
}

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

		client := &Client{
			conn: conn,
			game: s.game,
		}

		s.mu.Lock()
		s.connections[remoteAddr] = client
		s.mu.Unlock()

		s.game.AddPlayer(client)

		go client.handleRequest()
	}
}
