package net

import (
  "fmt"
  "log"
  "net"
)

type Server struct {
  host string
  port string
}

func (server *Server) Run() {
  listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", server.host, server.port))
  if err != nil {
    log.Fatal(err)
  }
  defer listener.Close()

	log.Printf("Listening on %s:%s", server.host, server.port);

  for {
    conn, err := listener.Accept()
    if err != nil {
      log.Fatal(err)
    }

		log.Printf("Accepted connection from %s", conn.RemoteAddr().String());

    client := &Client{
      conn: conn,
    }
    go client.handleRequest()
  }
}