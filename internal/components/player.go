package components

import (
	"dmud/internal/common"
	"strings"
	"sync"

	"github.com/rs/zerolog/log"
)

type PlayerComponent struct {
	sync.RWMutex
	Client common.Client
	Health *HealthComponent
	Name   string
	Room   *RoomComponent
	Target *PlayerComponent
}

///////////////////////////////////////////////////////////////////////////////////////////////
// ..
//

func (p *PlayerComponent) Broadcast(m string) {
	p.Client.SendMessage(m)
}

func (p *PlayerComponent) Look() {
	p.Broadcast(p.Room.Description)
}

func (p *PlayerComponent) Kill(target string) {
	if p.Target != nil {
		p.Broadcast("You are already attacking " + p.Target.Name)
		return
	}

	targetPlayer := p.Room.GetPlayer(target)
	if targetPlayer == nil {
		p.Broadcast("You don't see that here.")
		return
	}

	p.Target = targetPlayer

	log.Info().Msgf("Player %s is attacking %s", p.Name, p.Target.Name)
}

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

func (p *PlayerComponent) Say(msg string) {
	p.Room.Broadcast(p.Name + " says: " + msg)
}

func (p *PlayerComponent) Scan() {
	exits := []string{}
	for _, exit := range p.Room.Exits {
		exits = append(exits, exit.Direction)
	}
	p.Broadcast("Exits: " + strings.Join(exits, ", "))
}

func (p *PlayerComponent) Shout(msg string, depths ...int) {
	if p.Room == nil {
		p.Broadcast("You try to shout but it just comes out muffled.")
		return
	}
	log.Info().Msgf("Shout: %s", msg)

	depth := 10
	if len(depths) > 0 {
		depth = depths[0]
	}

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
		room.Broadcast(p.Name+" shouts: "+msg, p)
	}
}

func (p *PlayerComponent) Whisper(target *PlayerComponent, msg string) {
	target.Client.SendMessage(p.Name + " whispers: " + msg)
	p.Broadcast("You whisper: " + msg)
}

///////////////////////////////////////////////////////////////////////////////////////////////
// ..
//
