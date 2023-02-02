package main

import (
  "dmud/internal/net"
)

func main() {
  server := New(&Config{
    Host: "localhost",
    Port: "3333",
  })
  server.Run()
}