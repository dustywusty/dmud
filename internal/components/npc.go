package components

import (
	"dmud/internal/common"
	"math/rand"
	"sync"
	"time"
)

type NPC struct {
	sync.RWMutex

	Area              *Area
	Behavior          NPCBehavior
	Description       string
	Dialogue          []string
	Name              string
	LastAction        time.Time
	LastMovement      time.Time
	ConversationUntil time.Time
	Target            common.EntityID
	TemplateID        string
}

func (n *NPC) GetRandomDialogue() string {
	n.RLock()
	defer n.RUnlock()
	if len(n.Dialogue) == 0 {
		return ""
	}
	return n.Dialogue[rand.Intn(len(n.Dialogue))]
}

func (n *NPC) HoldConversation(duration time.Duration) {
	if duration <= 0 {
		return
	}
	until := time.Now().Add(duration)
	if until.IsZero() {
		return
	}

	n.Lock()
	if n.ConversationUntil.Before(until) {
		n.ConversationUntil = until
	}
	n.Unlock()
}

func (n *NPC) IsInConversation() bool {
	n.RLock()
	until := n.ConversationUntil
	n.RUnlock()
	return !until.IsZero() && time.Now().Before(until)
}
