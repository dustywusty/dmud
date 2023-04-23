package net

import (
  "fmt"
  "log"
  "net"

  "dmud/internal/game"
)

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
		server.game.AddPlayer(client)
		go client.handleRequest()
	}
}
