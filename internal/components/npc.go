package components

import (
	"dmud/internal/common"
	"math/rand"
	"sync"
	"time"
)

type NPC struct {
	sync.RWMutex

	Area         *Area
	Behavior     NPCBehavior
	Description  string
	Dialogue     []string
	Name         string
	LastAction   time.Time
	LastMovement time.Time
	Target       common.EntityID
	TemplateID   string
}

func (n *NPC) GetRandomDialogue() string {
	n.RLock()
	defer n.RUnlock()
	if len(n.Dialogue) == 0 {
		return ""
	}
	return n.Dialogue[rand.Intn(len(n.Dialogue))]
}
