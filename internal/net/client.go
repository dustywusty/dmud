package net

import (
	"bufio"
	"dmud/internal/ecs"
	"dmud/internal/game"
	"log"
	"net"
)

type Client struct {
	conn     net.Conn
	game     *game.Game
	EntityID ecs.EntityID
}

func (c *Client) RemoteAddr() string {
	return c.conn.RemoteAddr().String()
}

func (c *Client) SendMessage(msg string) {
	_, err := c.conn.Write([]byte("\b\b" + msg + "\n\n> "))
	if err != nil {
		return
	}
}

func (c *Client) CloseConnection() {
	err := c.conn.Close()
	if err != nil {
		return
	}
}

func (c *Client) handleRequest() {
	reader := bufio.NewReader(c.conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			c.conn.Close()
			//c.game.RemovePlayer(c.EntityID)
			return
		}
		//cmd := parseCommand(message)
		//if cmd != nil {
		//	log.Printf("Received command: %s, args: %s", cmd.Name, cmd.Arguments)
		//} else {
		//	log.Printf("Invalid command: %s", message)
		//}
		log.Printf("Received command: %s", message)
	}
}

//func parseCommand(message string) *Command {
//	words := strings.Fields(message)
//	if len(words) == 0 {
//		return nil
//	}
//
//	cmd := &Command{
//		Name:      strings.ToLower(words[0]),
//		Arguments: words[1:],
//	}
//
//	return cmd
//}
