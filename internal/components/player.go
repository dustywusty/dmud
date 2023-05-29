package components

import (
	"dmud/internal/common"
)

type PlayerComponent struct {
	Client common.Client
	Name   string
	Room   *RoomComponent
}

func (p *PlayerComponent) Say(msg string) {
	p.Room.MessageAllPlayers(p.Name+" says: "+msg, p)
}
