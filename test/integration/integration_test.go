package integration_test

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"dmud/internal/components"
	dmudnet "dmud/internal/net"

	"github.com/gorilla/websocket"
)

var testAddr string

func TestMain(m *testing.M) {
	// Locate project root relative to this source file.
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(filename), "..", "..")

	if err := os.Chdir(root); err != nil {
		panic("chdir to project root: " + err.Error())
	}

	if err := components.LoadNPCTemplates("./resources/npcs.json"); err != nil {
		panic("LoadNPCTemplates: " + err.Error())
	}
	components.InitializeQuests()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic("listen: " + err.Error())
	}
	testAddr = l.Addr().String()

	srv := dmudnet.NewServer(&dmudnet.ServerConfig{})
	srv.RunWithListener(l)

	// The spawn system checks every 5 s; wait 6 s so NPCs are present in areas.
	time.Sleep(6 * time.Second)

	code := m.Run()

	srv.Shutdown()
	os.Exit(code)
}

// ---------------------------------------------------------------------------
// testConn — lightweight WebSocket test helper
// ---------------------------------------------------------------------------

type testConn struct {
	conn *websocket.Conn
	msgs chan string // buffered; background goroutine feeds it
}

func dial(t *testing.T) *testConn {
	t.Helper()
	url := "ws://" + testAddr + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", url, err)
	}
	tc := &testConn{conn: conn, msgs: make(chan string, 128)}
	go func() {
		defer close(tc.msgs)
		for {
			_, p, err := conn.ReadMessage()
			if err != nil {
				return
			}
			tc.msgs <- string(p)
		}
	}()
	return tc
}

func (tc *testConn) send(t *testing.T, cmd string) {
	t.Helper()
	if err := tc.conn.WriteMessage(websocket.TextMessage, []byte(cmd)); err != nil {
		t.Fatalf("send %q: %v", cmd, err)
	}
}

// expect reads messages until one contains substr or the timeout fires.
// Returns the matching message.
func (tc *testConn) expect(t *testing.T, substr string, timeout time.Duration) string {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case msg, ok := <-tc.msgs:
			if !ok {
				t.Fatalf("connection closed while waiting for %q", substr)
			}
			if strings.Contains(msg, substr) {
				return msg
			}
		case <-deadline:
			t.Fatalf("timeout after %v waiting for %q", timeout, substr)
		}
	}
}

// drain collects all messages received within d.
func (tc *testConn) drain(d time.Duration) []string {
	var msgs []string
	timer := time.NewTimer(d)
	defer timer.Stop()
	for {
		select {
		case msg, ok := <-tc.msgs:
			if !ok {
				return msgs
			}
			msgs = append(msgs, msg)
		case <-timer.C:
			return msgs
		}
	}
}

func (tc *testConn) close() {
	_ = tc.conn.Close()
}

// ---------------------------------------------------------------------------
// parse helpers
// ---------------------------------------------------------------------------

// parseFirstExit extracts the first exit direction from look output containing
// "Exits: [north, south, ...]".
func parseFirstExit(msgs []string) string {
	for _, msg := range msgs {
		if i := strings.Index(msg, "Exits: ["); i >= 0 {
			rest := msg[i+len("Exits: ["):]
			if j := strings.IndexByte(rest, ']'); j >= 0 {
				exits := strings.Split(rest[:j], ", ")
				if len(exits) > 0 && exits[0] != "" {
					return strings.TrimSpace(exits[0])
				}
			}
		}
	}
	return ""
}

// parseNPCName extracts the first NPC name from lines of the form "<Name> is here."
func parseNPCName(msgs []string) string {
	for _, msg := range msgs {
		for _, line := range strings.Split(msg, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasSuffix(line, " is here.") {
				return strings.TrimSuffix(line, " is here.")
			}
		}
	}
	return ""
}

// waitForNPC polls look until an NPC appears in the area or timeout elapses.
// Returns the NPC name, or "" if none appeared in time.
func waitForNPC(t *testing.T, tc *testConn, timeout time.Duration) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		tc.send(t, "look")
		if name := parseNPCName(tc.drain(500 * time.Millisecond)); name != "" {
			return name
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// constants
// ---------------------------------------------------------------------------

const defaultTimeout = 3 * time.Second

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestConnect verifies that a fresh connection receives the welcome sequence
// (banner + area description).
func TestConnect(t *testing.T) {
	t.Parallel()
	tc := dial(t)
	defer tc.close()

	// The area description always includes an "Exits:" line.
	tc.expect(t, "Exits", defaultTimeout)
}

// TestLookCommand verifies the look command returns an area description.
func TestLookCommand(t *testing.T) {
	t.Parallel()
	tc := dial(t)
	defer tc.close()

	tc.expect(t, "Exits", defaultTimeout) // wait for welcome look
	tc.send(t, "look")
	tc.expect(t, "Exits", defaultTimeout)
}

// TestWhoCommand verifies the who command returns a player listing.
func TestWhoCommand(t *testing.T) {
	t.Parallel()
	tc := dial(t)
	defer tc.close()

	tc.expect(t, "Exits", defaultTimeout)
	tc.send(t, "who")
	// go-pretty renders headers in uppercase; the table always contains "PLAYER".
	tc.expect(t, "PLAYER", defaultTimeout)
}

// TestUnknownCommand verifies that unrecognised input produces an error message.
func TestUnknownCommand(t *testing.T) {
	t.Parallel()
	tc := dial(t)
	defer tc.close()

	tc.expect(t, "Exits", defaultTimeout)
	tc.send(t, "zzz")
	tc.expect(t, "What do you mean", defaultTimeout)
}

// TestSayCommand verifies that a message said by player A is received by
// player B in the same area.
func TestSayCommand(t *testing.T) {
	t.Parallel()
	a := dial(t)
	defer a.close()
	b := dial(t)
	defer b.close()

	// Ensure both players have entered the world (area description received).
	a.expect(t, "Exits", defaultTimeout)
	b.expect(t, "Exits", defaultTimeout)

	a.send(t, "say hello")
	b.expect(t, "hello", defaultTimeout)
}

// TestMoveCommand verifies that sending a direction moves the player and
// produces a new area description.
func TestMoveCommand(t *testing.T) {
	t.Parallel()
	tc := dial(t)
	defer tc.close()

	lookMsg := tc.expect(t, "Exits", defaultTimeout)
	dir := parseFirstExit([]string{lookMsg})
	if dir == "" {
		t.Fatal("no exit direction found in initial look output")
	}
	tc.send(t, dir)
	tc.expect(t, "Exits", defaultTimeout) // new area description
}

// TestCastHeal verifies the heal spell reports health information.
func TestCastHeal(t *testing.T) {
	t.Parallel()
	tc := dial(t)
	defer tc.close()

	tc.expect(t, "Exits", defaultTimeout)
	tc.send(t, "cast heal")
	tc.expect(t, "health", defaultTimeout)
}

// TestCastCharm verifies the charm spell produces a response containing "charm".
// Skips if no NPC has spawned in the starting area yet.
func TestCastCharm(t *testing.T) {
	t.Parallel()
	tc := dial(t)
	defer tc.close()

	tc.expect(t, "Exits", defaultTimeout)

	npcName := waitForNPC(t, tc, 10*time.Second)
	if npcName == "" {
		t.Skip("no NPC in starting area; skipping charm test")
	}

	tc.send(t, "cast charm "+npcName)
	tc.expect(t, "charm", defaultTimeout)
}

// TestKillNPC verifies that attacking NPCs produces a combat message.
// Skips if no NPC has spawned in the starting area.
func TestKillNPC(t *testing.T) {
	t.Parallel()
	tc := dial(t)
	defer tc.close()

	tc.expect(t, "Exits", defaultTimeout)

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		tc.send(t, "kill all")
		for _, msg := range tc.drain(500 * time.Millisecond) {
			if strings.Contains(msg, "attacks") {
				return // pass: attack message received
			}
		}
	}
	t.Skip("no NPCs found in starting area within timeout")
}

// TestRecall verifies the recall command returns the player to the starting area.
func TestRecall(t *testing.T) {
	t.Parallel()
	tc := dial(t)
	defer tc.close()

	tc.expect(t, "Exits", defaultTimeout)
	tc.send(t, "recall")
	tc.expect(t, "starting area", defaultTimeout)
}
