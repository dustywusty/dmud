package components

import (
	"dmud/internal/common"
	"sync"
)

type Player struct {
	sync.RWMutex

	Client common.Client

	Name string
	Room *Room
}

func (p *Player) Broadcast(m string) {
	p.Client.SendMessage(m)
}

func (p *Player) Look() {
	p.Broadcast(p.Room.Description)
}
