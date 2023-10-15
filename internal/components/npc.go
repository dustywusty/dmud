package components

import (
	"sync"
)

type NPC struct {
	sync.RWMutex

	Name string
	Room *Room
}
