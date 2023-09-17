package components

import (
	"dmud/internal/common"
	"sync"
)

type PlayerComponent struct {
	sync.RWMutex
	Client common.Client
	Name   string
	Room   *RoomComponent
	Target *PlayerComponent
}

// -----------------------------------------------------------------------------

func (p *PlayerComponent) Broadcast(m string) {
	p.Client.SendMessage(m)
}

func (p *PlayerComponent) Look() {
	p.Broadcast(p.Room.Description)
}

// func (p *PlayerComponent) Kill(target string) {
// 	if p.Target != nil {
// 		p.Broadcast("You are already attacking " + p.Target.Name)
// 		return
// 	}

// 	targetPlayer := p.Room.GetPlayer(target)
// 	if targetPlayer == nil {
// 		p.Broadcast("You don't see that here.")
// 		return
// 	}

// 	p.Target = targetPlayer

// 	log.Info().Msgf("Player %s is attacking %s", p.Name, p.Target.Name)
// }

func (p *PlayerComponent) Move(direction string) {
	exit := p.Room.GetExit(direction)
	if exit == nil {
		p.Broadcast("You can't go that way.")
		return
	}

	p.Room.RemovePlayer(p)
	p.Room = exit.Room
	p.Room.AddPlayer(p)

	p.Broadcast(p.Room.Description)
}

// func (p *PlayerComponent) Whisper(target *PlayerComponent, msg string) {
// 	target.Client.SendMessage(p.Name + " whispers: " + msg)
// 	p.Broadcast("You whisper: " + msg)
// }
