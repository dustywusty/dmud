package ecs

import (
	"os"
	"runtime"
	"testing"
	"time"

	"dmud/internal/common"
	"dmud/internal/components"

	"github.com/rs/zerolog"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

// newTestWorld builds an empty World without going through NewWorld(), so the
// test does not depend on ./resources/areas.json being present.
func newTestWorld() *World {
	return &World{
		entities:   make(map[common.EntityID]Entity),
		components: make(map[common.EntityID]map[string]Component),
	}
}

// NOTE: TestWorld_CombatTargetID_DataRace used to live here. It demonstrated
// the mismatched-locking bug on Combat.TargetID (unlocked writes in
// systems/combat.go vs RLock reads in systems/ai.go). That bug is now fixed at
// the source: Combat no longer carries a mutex, and it is only ever touched on
// the game loop goroutine. The actor-model invariant is covered end-to-end by
// TestGameLoop_ConcurrentClients_NoRace in internal/game, which runs the real
// loop (connects, commands, disconnects, NPC combat) clean under -race.

// TestWorld_LockOrderDeadlock reproduces the lock-order inversion baked into
// World. Three lock-acquisition orders all exist in the real code:
//
//	(A) world read held across a per-object lock:
//	    FindEntitiesByComponentPredicate holds componentMutex.RLock() while the
//	    predicate runs, and real predicates take per-object locks
//	    (components/area.go:54-63 -> npc.RLock()).
//
//	(B) per-object lock held across a world read:
//	    any code that holds a per-object lock (npc.Lock(), player.Lock(), ...)
//	    while calling into the World, which takes componentMutex.RLock().
//	    This hazard shrinks as components shed their mutexes (Combat and Health
//	    already have) and disappears once none remain. NPC still has one, so it
//	    stands in below.
//
//	(C) a componentMutex writer:
//	    RemoveComponent / AddComponent / spawns take componentMutex.Lock()
//	    constantly (e.g. systems/combat.go:49).
//
// Go's sync.RWMutex blocks NEW readers once a writer is waiting (to avoid
// writer starvation). That turns (A)+(B)+(C) into a cycle:
//
//	G1 holds componentMutex.RLock, waits for the object lock G2 holds.
//	G2 holds the object lock, waits for componentMutex.RLock (blocked by C).
//	G3 (writer) waits for componentMutex.Lock (blocked by G1's RLock).
//
// The test forces that interleaving deterministically and FAILS with a
// goroutine dump if the operations don't complete — i.e. if they deadlock.
//
// Guarded by an env var because, when the deadlock fires, the three worker
// goroutines park permanently (they leak until the test binary exits). Run with:
//
//	DMUD_DEADLOCK_DEMO=1 go test ./internal/ecs -run Deadlock -v
func TestWorld_LockOrderDeadlock(t *testing.T) {
	if os.Getenv("DMUD_DEADLOCK_DEMO") == "" {
		t.Skip("set DMUD_DEADLOCK_DEMO=1 to run the lock-order deadlock reproduction (it parks goroutines on purpose)")
	}

	w := newTestWorld()
	e := NewEntity()
	w.AddEntity(e)
	// NPC still carries a sync.RWMutex (taken inside Area.GetNPCs' predicate),
	// so it stands in for "a component that hasn't shed its mutex yet".
	npc := &components.NPC{Name: "deadlock-dummy"}
	w.AddComponent(&e, npc)

	g1Holds := make(chan struct{})   // G1 now holds componentMutex.RLock
	g1Proceed := make(chan struct{}) // release G1 to grab the object lock
	done := make(chan struct{})

	// G1 — order (A): world read held across a per-object lock.
	// Mirrors Area.GetNPCs, whose predicate calls npc.RLock().
	go func() {
		_, _ = w.FindEntitiesByComponentPredicate("NPC", func(i interface{}) bool {
			// We are inside the predicate, so componentMutex.RLock() is held.
			close(g1Holds)
			<-g1Proceed
			n := i.(*components.NPC)
			n.Lock() // wants the object lock that G2 holds -> blocks
			n.Unlock()
			return true
		})
	}()

	// G2 — order (B): per-object lock held across a world read.
	go func() {
		<-g1Holds  // ensure G1 holds componentMutex.RLock first
		npc.Lock() // hold the object's write lock

		// G3 — order (C): a pending componentMutex writer. RemoveComponent takes
		// ONLY componentMutex.Lock() (no entityMutex), so it parks as a pending
		// writer behind G1's RLock and makes G2's upcoming RLock block.
		go func() {
			w.RemoveComponent("does-not-exist", "NPC")
		}()
		time.Sleep(50 * time.Millisecond) // let G3 register as a pending writer

		close(g1Proceed)                   // release G1; it now blocks on npc.Lock()
		_, _ = w.GetComponent(e.ID, "NPC") // wants componentMutex.RLock() -> blocked by pending G3
		npc.Unlock()
		close(done)
	}()

	select {
	case <-done:
		// Completed: no deadlock on this run.
	case <-time.After(5 * time.Second):
		buf := make([]byte, 1<<16)
		n := runtime.Stack(buf, true)
		t.Fatalf("DEADLOCK: world operations did not complete within 5s.\n\n"+
			"Parked goroutines (look for FindEntitiesByComponentPredicate, GetComponent, RemoveComponent\n"+
			"all blocked on sync.RWMutex):\n\n%s", buf[:n])
	}
}
