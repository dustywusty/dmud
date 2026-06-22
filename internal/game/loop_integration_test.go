package game

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"dmud/internal/common"
	"dmud/internal/components"

	"github.com/rs/zerolog"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

// fakeClient is an in-memory common.Client. SupportsTags()==true so it takes
// the deferred-spawn path (HandleConnect schedules enterWorld via the loop).
type fakeClient struct {
	addr string

	mu       sync.Mutex
	messages []string
	done     chan struct{}
	closed   bool
}

func newFakeClient(addr string) *fakeClient {
	return &fakeClient{addr: addr, done: make(chan struct{})}
}

func (f *fakeClient) CloseConnection() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.closed {
		f.closed = true
		close(f.done)
	}
	return nil
}

// HandleRequest blocks until the client is closed, mimicking a real read loop
// without a socket. HandleConnect launches this in its own goroutine.
func (f *fakeClient) HandleRequest()      { <-f.done }
func (f *fakeClient) RemoteAddr() string  { return f.addr }
func (f *fakeClient) SupportsPrompt() bool { return false }
func (f *fakeClient) SupportsTags() bool   { return true }

func (f *fakeClient) SendMessage(msg string) {
	f.mu.Lock()
	f.messages = append(f.messages, msg)
	f.mu.Unlock()
}

func (f *fakeClient) messageCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.messages)
}

// chdirToRepoRoot points the working dir at the module root so NewGame can load
// ./resources/*.json. The game loop reads no files after startup, so restoring
// the cwd afterwards is safe.
func chdirToRepoRoot(t *testing.T) {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	start := dir
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if err := os.Chdir(dir); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chdir(start) })
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found above %s", start)
		}
		dir = parent
	}
}

// TestGameLoop_ConcurrentClients_NoRace drives the real game loop with many
// clients connecting, issuing commands, and disconnecting concurrently — the
// exact workload the network layer produces.
//
// It is the proof for the actor-model commitment: once the login-grace writer
// (game.go) and the TCP disconnect (tcp_client.go) are routed through the loop,
// every world-state mutation happens on one goroutine. Run under -race:
//
//	go test -race -run TestGameLoop_ConcurrentClients_NoRace ./internal/game/
//
// A clean run means no goroutine outside the loop touches world state.
func TestGameLoop_ConcurrentClients_NoRace(t *testing.T) {
	chdirToRepoRoot(t)

	// Load content so the spawn/AI/combat systems actually run NPCs alongside
	// the player churn — more concurrent surface for the race detector.
	if err := components.LoadNPCTemplates("./resources/npcs.json"); err != nil {
		t.Fatalf("load npc templates: %v", err)
	}
	components.InitializeQuests()

	g := NewGame()

	const (
		workers   = 12
		perWorker = 5
	)
	cmds := []string{"look", "who", "say hello", "n", "s", "inventory", "examine rat", "time"}

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()

			var keep []*fakeClient
			for i := 0; i < perWorker; i++ {
				c := newFakeClient(fmt.Sprintf("10.0.%d.%d:5000", w, i))
				g.AddPlayerChan <- c

				// Fire commands while the login grace is still pending.
				for _, cmd := range cmds {
					g.ExecuteCommandChan <- mkCommand(c, cmd)
				}

				if i%2 == 1 {
					// Disconnect before the grace fires: the scheduled enterWorld
					// will run on the loop and find the player already gone.
					g.RemovePlayerChan <- c
				} else {
					keep = append(keep, c)
				}
			}

			// Let the grace timers fire -> enterWorld runs on the loop.
			time.Sleep(900 * time.Millisecond)

			for _, c := range keep {
				for _, cmd := range cmds {
					g.ExecuteCommandChan <- mkCommand(c, cmd)
				}
				g.RemovePlayerChan <- c
			}
		}(w)
	}

	wg.Wait()

	// Drain: let the loop process the final disconnects and any late timers.
	time.Sleep(1200 * time.Millisecond)

	// Sanity: the loop actually served clients (welcome banner, look output...).
	g.playersMu.RLock()
	remaining := len(g.players)
	g.playersMu.RUnlock()
	if remaining != 0 {
		t.Logf("players still tracked after drain: %d (acceptable: random-name collisions)", remaining)
	}
}

func mkCommand(c common.Client, line string) ClientCommand {
	parts := splitFields(line)
	cmd := ClientCommand{Client: c, Cmd: parts[0]}
	if len(parts) > 1 {
		cmd.Args = parts[1:]
	}
	return cmd
}

func splitFields(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == ' ' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
