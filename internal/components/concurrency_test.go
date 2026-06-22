package components

import (
	"sync"
	"testing"

	"github.com/rs/zerolog"
)

func init() {
	// Keep the race/deadlock output readable.
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

// NOTE: TestHealth_ConcurrentAccess_DataRace used to live here. It demonstrated
// TODO.md "Critical Concurrency Issue #1" — Health embedded sync.RWMutex but
// Heal()/TakeDamage() mutated Current/Status without taking it. That bug is now
// fixed: Health carries no mutex and is only touched on the game loop goroutine.
// The actor-model invariant is verified end-to-end by
// TestGameLoop_ConcurrentClients_NoRace (internal/game), which runs the real
// loop clean under -race.

// TestExperience_ConcurrentAccess_Safe is the contrast case. Experience
// (experience.go) takes its own lock inside AddXP/GetLevel/GetCurrent, so the
// identical access pattern is race-free.
//
// PASSES under both `go test` and `go test -race`.
//
// The only difference between this and the Health test above is that this type
// actually uses the mutex it embeds. That is the whole fix for #1.
func TestExperience_ConcurrentAccess_Safe(t *testing.T) {
	e := NewExperience()

	var wg sync.WaitGroup

	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 2000; j++ {
				e.AddXP(10)
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for j := 0; j < 8000; j++ {
			_ = e.GetLevel()
			_ = e.GetCurrent()
		}
	}()

	wg.Wait()
}
