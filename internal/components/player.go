package components

import (
	"dmud/internal/common"
)

type PlayerComponent struct {
	Client common.Client
	Name   string
	Room   *RoomComponent
}

///////////////////////////////////////////////////////////////////////////////////////////////
// Public
//

func (p *PlayerComponent) Say(msg string) {
	p.Room.MessageAllPlayers(p.Name+" says: "+msg, p)
	p.Client.SendMessage("You say: " + msg)
}

func (p *PlayerComponent) Shout(msg string, depth int) {
	visited := make(map[*RoomComponent]bool)
	queue := []*RoomComponent{p.Room}

	for depth > 0 && len(queue) > 0 {
		depth--
		nextQueue := []*RoomComponent{}

		for _, room := range queue {
			visited[room] = true
			for _, exit := range room.Exits {
				if !visited[exit.Room] {
					visited[exit.Room] = true
					nextQueue = append(nextQueue, exit.Room)
				}
			}
		}
		queue = nextQueue
	}

	for room := range visited {
		room.MessageAllPlayers(p.Name+" shouts: "+msg, p)
	}
	p.Client.SendMessage("You shout: " + msg)
}

func (p *PlayerComponent) Whisper(target *PlayerComponent, msg string) {
	target.Client.SendMessage(p.Name + " whispers: " + msg)
	p.Client.SendMessage("You whisper: " + msg)
}
