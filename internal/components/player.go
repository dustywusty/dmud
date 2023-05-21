package components

import "dmud/internal/common"

type PlayerComponent struct {
	Client common.Client
	Room   *RoomComponent

	name string
}

func (p *PlayerComponent) Name() string {
	return p.name
}

func (p *PlayerComponent) SetName(name string) {
	p.name = name
}
