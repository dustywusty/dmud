package game

import (
	"testing"
	"time"
)

// TestHandleKillNoDeadlock guards against the self-deadlock where HandleKill held
// g.playersMu as a WRITE lock and then called playerMeleeDamage -> getPlayerEntity,
// which re-locks g.playersMu for reading. Go's RWMutex is not reentrant, so that
// froze the game loop. With a read lock the re-entrant read is fine.
func TestHandleKillNoDeadlock(t *testing.T) {
	r := newSpellRig(t, 5)

	done := make(chan struct{})
	go func() {
		r.g.HandleKill(r.player, "goblin") // finds the goblin -> reaches playerMeleeDamage
		close(done)
	}()

	select {
	case <-done:
		// returned promptly — no deadlock
	case <-time.After(3 * time.Second):
		t.Fatal("HandleKill deadlocked: a write lock on playersMu was held across a read-locking call")
	}
}
