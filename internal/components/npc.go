package components

import (
	"dmud/internal/common"
	"math/rand"
	"time"
)

// NPC carries no mutex: all NPC state is read and written exclusively on the
// game-loop goroutine (AI/combat/status systems via world.Update, and command
// handlers, all of which run on the loop), so the actor model already
// serializes access. See the actor-model commitment proven by
// TestGameLoop_ConcurrentClients_NoRace.
type NPC struct {
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

	if n.ConversationUntil.Before(until) {
		n.ConversationUntil = until
	}
}

func (n *NPC) IsInConversation() bool {
	until := n.ConversationUntil
	return !until.IsZero() && time.Now().Before(until)
}
