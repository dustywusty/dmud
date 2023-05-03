package net

import (
  "dmud/internal/game"
  "fmt"
  "log"
  "net"
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

func (server *Server) Run() {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", server.host, server.port))
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()

	log.Printf("Listening on %s:%s", server.host, server.port)

	server.game = game.NewGame()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Accepted connection from %s", conn.RemoteAddr().String())

		client := &Client{
			conn: conn,
			game: server.game,
			name: "",
		}
		client.name = client.generateRandomName()

		client.SendMessage(fmt.Sprintf("Welcome to the server, %s!", client.name))

		server.game.AddPlayer(client)

		go client.handleRequest()
	}
}
