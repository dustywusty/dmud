package net

import (
	"fmt"
	"log"
	"net"

	"dmud/internal/game"
)

type ServerConfig struct {
	Host string
	Port string
}

func NewServer(config *ServerConfig) *Server {
	return &Server{
		host: config.Host,
		port: config.Port,
	}
}

type Server struct {
	host        string
	port        string
	connections map[string]*Client
	game        *game.Game
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

		log.Printf("Accepted connection from %s", conn.RemoteAddr().String())

		client := &Client{
			conn: conn,
			game: s.game,
		}

		s.game.AddPlayerChan <- client

		go client.handleRequest()
	}
}
