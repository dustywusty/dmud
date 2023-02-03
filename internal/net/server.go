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

  for {
    conn, err := listener.Accept()
    if err != nil {
      log.Fatal(err)
    }

    client := &Client{
      conn: conn,
    }
    go client.handleRequest()
  }
}