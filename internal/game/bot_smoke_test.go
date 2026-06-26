package game

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"dmud/internal/common"
	"dmud/internal/components"

	"github.com/rs/zerolog"
)

func init() {
	// Keep the loop's structured logging out of test output.
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

// smokeClient is a self-contained in-memory common.Client used to drive the
// game loop without a real socket. SupportsTags()==true takes the deferred
// spawn path (HandleConnect schedules enterWorld via the loop).
type smokeClient struct {
	addr string

	mu       sync.Mutex
	messages []string
	done     chan struct{}
	closed   bool
}

func newSmokeClient(addr string) *smokeClient {
	return &smokeClient{addr: addr, done: make(chan struct{})}
}

func (c *smokeClient) CloseConnection() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		close(c.done)
	}
	return nil
}

// HandleRequest blocks until closed, mimicking a real read loop without a
// socket. HandleConnect launches this in its own goroutine.
func (c *smokeClient) HandleRequest()       { <-c.done }
func (c *smokeClient) RemoteAddr() string   { return c.addr }
func (c *smokeClient) SupportsPrompt() bool { return false }
func (c *smokeClient) SupportsTags() bool   { return true }

func (c *smokeClient) SendMessage(msg string) {
	c.mu.Lock()
	c.messages = append(c.messages, msg)
	c.mu.Unlock()
}

func (c *smokeClient) messageCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.messages)
}

func (c *smokeClient) snapshot() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, len(c.messages))
	copy(out, c.messages)
	return out
}

func (c *smokeClient) hasMessageContaining(substr string) bool {
	for _, m := range c.snapshot() {
		if strings.Contains(m, substr) {
			return true
		}
	}
	return false
}

// smokeChdir points the working dir at the module root so NewGame can load
// ./resources/*.json. The game loop reads no files after startup, so restoring
// the cwd afterwards is safe.
func smokeChdir(t *testing.T) {
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

// smokeCmd builds a ClientCommand from a raw command line.
func smokeCmd(c common.Client, line string) ClientCommand {
	parts := strings.Fields(line)
	cmd := ClientCommand{Client: c, Cmd: parts[0]}
	if len(parts) > 1 {
		cmd.Args = parts[1:]
	}
	return cmd
}

// waitFor polls cond until it returns true or the timeout elapses. The game
// loop processes channels asynchronously, so behavioural assertions have to
// wait for the loop to catch up rather than read state immediately.
func waitFor(cond func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return cond()
}

// TestServer_BotsSmoke stands up the real game loop, connects a handful of
// "bots" (in-memory clients), and has them walk around and interact. It
// validates *observable behaviour*: spawning, chat, movement, and loop
// liveness under concurrent load — a fast end-to-end confidence check that the
// server actually does something sensible.
//
//	go test -v -run TestServer_BotsSmoke ./internal/game/
func TestServer_BotsSmoke(t *testing.T) {
	smokeChdir(t)

	if err := components.LoadNPCTemplates("./resources/npcs.json"); err != nil {
		t.Fatalf("load npc templates: %v", err)
	}
	components.InitializeQuests()

	g := NewGame()

	const numBots = 6
	bots := make([]*smokeClient, numBots)
	for i := range bots {
		b := newSmokeClient(fmt.Sprintf("172.16.0.%d:6000", i))
		bots[i] = b
		g.AddPlayerChan <- b
	}

	// 1) Every bot enters the world (deferred-spawn login grace is ~750ms) and
	//    receives its initial room description.
	for i, b := range bots {
		if !waitFor(func() bool { return b.messageCount() > 0 }, 3*time.Second) {
			t.Fatalf("bot %d never spawned into the world", i)
		}
	}

	// New characters spawn as un-manifested ghosts; this test exercises
	// post-creation play, so manifest the bots before they act.
	g.playersMu.RLock()
	for _, ent := range g.players {
		if pc, err := g.world.GetComponent(ent.ID, "Player"); err == nil {
			if p, ok := pc.(*components.Player); ok {
				p.Created = true
			}
		}
		g.world.RemoveComponent(ent.ID, "Creation")
	}
	g.playersMu.RUnlock()

	// 2) Chat propagates between co-located bots: they all start in the default
	//    area, so a `say` from one must reach the others. These bots are
	//    tag-capable, so the speech arrives as a structured comms event with the
	//    words in its JSON "text" field (plain-text clients get "X says: …").
	const phrase = "smoke-test-marker"
	g.ExecuteCommandChan <- smokeCmd(bots[0], "say "+phrase)
	heard := waitFor(func() bool {
		for _, b := range bots[1:] {
			if b.hasMessageContaining(`"text":"` + phrase + `"`) {
				return true
			}
		}
		return false
	}, 2*time.Second)
	if !heard {
		t.Error("no other bot heard the chat message — area broadcast may be broken")
	}

	// 3) Movement actually relocates a bot. Walk a real exit out of the default
	//    area and confirm the destination room's description arrives.
	if len(g.defaultArea.Exits) > 0 {
		exit := g.defaultArea.Exits[0]
		dest := strings.TrimSpace(exit.Area.Description)
		mover := bots[1]
		g.ExecuteCommandChan <- smokeCmd(mover, exit.Direction)
		if dest != "" {
			if !waitFor(func() bool { return mover.hasMessageContaining(dest) }, 2*time.Second) {
				t.Errorf("bot did not receive destination room description after moving %q", exit.Direction)
			}
		}
	}

	// 4) Concurrent random walk: every bot fires a stream of commands at once,
	//    exercising movement, combat lookups, chat, and queries together.
	actions := []string{
		"look", "who", "say hi", "north", "south", "east", "west",
		"up", "down", "examine rat", "kill rat", "inventory", "time",
	}
	var wg sync.WaitGroup
	for _, b := range bots {
		wg.Add(1)
		go func(b *smokeClient) {
			defer wg.Done()
			for n := 0; n < 40; n++ {
				g.ExecuteCommandChan <- smokeCmd(b, actions[rand.Intn(len(actions))])
			}
		}(b)
	}
	wg.Wait()

	// 5) Liveness: after all that churn the loop must still answer. Probe with a
	//    `say` carrying a unique nonce and wait for that exact echo. A plain
	//    message-count bump is not enough: ExecuteCommandChan is buffered, so the
	//    loop may still be draining the churn backlog when we send this — an
	//    earlier command's response could satisfy the wait before the probe is
	//    ever processed. Echoing a unique string proves the loop reached *this*
	//    command, not a stale one still in the queue.
	const probe = "liveness-probe-9f3a2b"
	g.ExecuteCommandChan <- smokeCmd(bots[0], "say "+probe)
	if !waitFor(func() bool { return bots[0].hasMessageContaining(`"text":"` + probe + `"`) }, 2*time.Second) {
		t.Fatal("game loop stopped responding after the bot churn (possible deadlock)")
	}

	// 6) Clean disconnect: every bot leaves and the player table empties out.
	for _, b := range bots {
		g.RemovePlayerChan <- b
	}
	emptied := waitFor(func() bool {
		g.playersMu.RLock()
		defer g.playersMu.RUnlock()
		return len(g.players) == 0
	}, 3*time.Second)
	if !emptied {
		g.playersMu.RLock()
		remaining := len(g.players)
		g.playersMu.RUnlock()
		t.Errorf("players still tracked after disconnect: %d", remaining)
	}

	// Under `go test -v`, dump what each bot actually saw so you can watch the
	// run instead of just reading pass/fail.
	if testing.Verbose() {
		for i, b := range bots {
			msgs := b.snapshot()
			t.Logf("=== bot %d (%s) received %d messages ===", i, b.addr, len(msgs))
			for _, m := range msgs {
				t.Logf("  | %s", strings.ReplaceAll(strings.TrimRight(m, "\n"), "\n", "\n  | "))
			}
		}
	}
}
