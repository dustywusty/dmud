package net

import (
  "fmt"
  "log"
  "net"

  "dmud/internal/game"
)

// Server represents the MUD server, handling client connections and managing the game state.
type Server struct {
  host        string
  port        string
  connections map[string]*Client
  game        *game.Game
}

// Run starts the server, listens for incoming connections, and manages client requests.
func (server *Server) Run() {
  // Set up the server to listen for incoming connections.
  listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", server.host, server.port))
  if err != nil {
    log.Fatal(err)
  }
  defer listener.Close()

  log.Printf("Listening on %s:%s", server.host, server.port)

  // Initialize the game state.
  server.game = game.NewGame()

  // Continuously accept new connections and handle client requests.
  for {
    conn, err := listener.Accept()
    if err != nil {
      log.Fatal(err)
    }

    // Log the accepted connection.
    log.Printf("Accepted connection from %s", conn.RemoteAddr().String())

    // Create a new client and assign a random name.
    client := &Client{
      conn: conn,
      game: server.game,
      name: "",
    }
    client.name = client.generateRandomName()

    // Add the client to the game and handle its requests in a separate goroutine.
    server.game.AddPlayer(client)
    go client.handleRequest()
  }
}
