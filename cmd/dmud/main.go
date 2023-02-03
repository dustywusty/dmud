package main

import (
  "dmud/internal/net"
)

func main() {
  server := net.NewServer(&net.ServerConfig{
    Host: "localhost",
    Port: "3333",
  })
  server.Run()
}