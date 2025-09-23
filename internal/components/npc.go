// internal/components/npc.go
package components

import (
	"dmud/internal/common"
	"math/rand"
	"sync"
	"time"
)

type NPC struct {
	sync.RWMutex

	Name        string
	Description string
	Area        *Area
	TemplateID  string
	Behavior    NPCBehavior
	Dialogue    []string
	LastAction  time.Time
	Target      common.EntityID // For aggressive NPCs
}

func (n *NPC) GetRandomDialogue() string {
	n.RLock()
	defer n.RUnlock()

	if len(n.Dialogue) == 0 {
		return ""
	}

	return n.Dialogue[rand.Intn(len(n.Dialogue))]
}
