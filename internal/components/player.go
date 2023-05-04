package components

import (
	"dmud/internal/net"
	"fmt"
	"math/rand"
)

type PlayerComponent struct {
	Client *net.Client
	name   string
}

func (p *PlayerComponent) Name() string {
	return p.name
}

func (p *PlayerComponent) SetName(name string) {
	p.name = name
}

func (p *PlayerComponent) GenerateRandomName() {
	p.name = fmt.Sprintf("Guest%d", rand.Intn(10000))
}
