package net

import (
  "bufio"
	"log"
  "net"
)

type Client struct {
  conn net.Conn
}

func (client *Client) handleRequest() {
	client.conn.Write([]byte("Welcome to the server!\n\n> "));
  reader := bufio.NewReader(client.conn)
  for {
    message, err := reader.ReadString('\n')
    if err != nil {
      client.conn.Close()
      return
    }
		log.Printf("Message incoming: %s", string(message));
    client.conn.Write([]byte("> "))
  }
}