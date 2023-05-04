package net

import (
	game2 "dmud/internal/game"
	"fmt"
	"log"
	"net"

	"dmud/internal/components"
	"dmud/internal/ecs"
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
	game        *game2.Game
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

	s.game = game2.NewGame()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Accepted connection from %s", conn.RemoteAddr().String())

		client := &Client{
			conn: conn,
		}

		playerEntity := ecs.NewEntity()
		playerComponent := components.PlayerComponent{
			Client: client,
		}

		s.game.World.AddEntity(playerEntity)
		s.game.World.AddComponent(playerEntity, &playerComponent)

		s.game.AddPlayer(playerEntity.ID)

		go client.handleRequest()
	}
}
