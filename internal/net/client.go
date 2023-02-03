package net

import (
  "bufio"
  "fmt"
  "net"
)

type Client struct {
  conn net.Conn
}

func (client *Client) handleRequest() {
  reader := bufio.NewReader(client.conn)
  for {
    message, err := reader.ReadString('\n')
    if err != nil {
      client.conn.Close()
      return
    }
    fmt.Printf("Message incoming: %s", string(message))
    client.conn.Write([]byte("Message received.\n"))
  }
}