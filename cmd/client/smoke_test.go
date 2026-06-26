package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestLiveServerSmoke connects to a real dmud server and verifies the transport
// and classifier against live output. It is skipped unless DMUD_SMOKE=1 and a
// server is reachable (default ws://localhost:8080/ws, override with DMUD_ADDR).
//
//	make dev            # in one terminal
//	DMUD_SMOKE=1 go test ./cmd/client/ -run TestLiveServerSmoke -v
func TestLiveServerSmoke(t *testing.T) {
	if os.Getenv("DMUD_SMOKE") != "1" {
		t.Skip("set DMUD_SMOKE=1 to run the live server smoke test")
	}

	addr := os.Getenv("DMUD_ADDR")
	if addr == "" {
		addr = "localhost:8080"
	}

	c := newConn(transportWS, addr)
	if err := c.dial(); err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	defer c.Close()

	// Give the deferred-spawn grace period time to drop us into the world, then
	// look around and take a step north so the auto-mapper has an edge to draw.
	mp := newMapper()
	time.Sleep(1 * time.Second)
	if err := c.Send("look"); err != nil {
		t.Fatalf("send look: %v", err)
	}
	time.AfterFunc(1500*time.Millisecond, func() {
		mp.noteCommand("north")
		_ = c.Send("north")
	})

	var gotRoom, gotState, gotMain bool
	deadline := time.After(4 * time.Second)
	for {
		select {
		case ev := <-c.events:
			chunk, ok := ev.(chunkMsg)
			if !ok {
				continue
			}
			r := classify(chunk.raw)
			switch {
			case r.status != nil:
				gotState = true
			case r.room != nil:
				gotRoom = true
				mp.arrive(r.room.Title, r.room.Exits)
			case r.toMain != "":
				gotMain = true
			}
			t.Logf("chunk: %q", truncate(chunk.raw, 80))
		case <-deadline:
			t.Logf("results: room=%v state=%v main=%v mappedRooms=%d", gotRoom, gotState, gotMain, len(mp.rooms))
			for title, r := range mp.rooms {
				t.Logf("  mapped: %q at %+v", title, r.pos)
			}
			if !gotMain {
				t.Errorf("received no main output from server")
			}
			if !gotRoom && !gotState {
				t.Errorf("expected at least a room block or STATE frame; got neither")
			}
			if len(mp.rooms) < 1 {
				t.Errorf("auto-mapper recorded no rooms")
			}
			return
		}
	}
}

// TestLiveEventSmoke verifies the server pushes EVENT|room.contents: just
// connecting (which drops you into a room) should yield a contents event within
// a couple seconds, with no look required.
//
//	cd ../../  (a server with the EVENT protocol)  PORT=8099 ./bin/dmud &
//	DMUD_SMOKE=1 DMUD_ADDR=localhost:8099 go test ./cmd/client/ -run TestLiveEventSmoke -v
func TestLiveEventSmoke(t *testing.T) {
	if os.Getenv("DMUD_SMOKE") != "1" {
		t.Skip("set DMUD_SMOKE=1 to run the live event smoke test")
	}
	addr := os.Getenv("DMUD_ADDR")
	if addr == "" {
		addr = "localhost:8080"
	}

	c := newConn(transportWS, addr)
	if err := c.dial(); err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	defer c.Close()

	var gotContents, gotVitals, gotLegacyState bool
	deadline := time.After(4 * time.Second)
	for {
		select {
		case ev := <-c.events:
			chunk, ok := ev.(chunkMsg)
			if !ok {
				continue
			}
			if strings.HasPrefix(chunk.raw, "STATE|") {
				gotLegacyState = true
			}
			r := classify(chunk.raw)
			if r.contents != nil {
				gotContents = true
				t.Logf("room.contents: players=%v npcs=%v items=%v corpses=%v",
					r.contents.Players, r.contents.NPCs, r.contents.Items, r.contents.Corpses)
			}
			if r.status != nil && strings.HasPrefix(chunk.raw, "EVENT|") {
				gotVitals = true
				t.Logf("char.vitals: hp=%d/%d level=%d xp=%d/%d area=%q effects=%v",
					r.status.HP, r.status.MaxHP, r.status.Level, r.status.XP, r.status.ReqXP, r.status.Area, r.status.Effects)
			}
		case <-deadline:
			if !gotContents || !gotVitals {
				t.Errorf("expected both events within 4s (room.contents=%v char.vitals=%v)", gotContents, gotVitals)
			}
			if gotLegacyState {
				t.Errorf("legacy STATE| frame should be retired, but one arrived")
			}
			return
		}
	}
}

// TestLiveIdentitySmoke confirms a persistence-enabled server hands the client
// an IDENTITY| login id on connect (which the client stores and replays).
//
//	cd ../../  DMUD_PERSISTENCE=file PORT=8099 ./bin/dmud &
//	DMUD_SMOKE=1 DMUD_ADDR=localhost:8099 go test ./cmd/client/ -run TestLiveIdentitySmoke -v
func TestLiveIdentitySmoke(t *testing.T) {
	if os.Getenv("DMUD_SMOKE") != "1" {
		t.Skip("set DMUD_SMOKE=1 to run the live identity smoke test")
	}
	addr := os.Getenv("DMUD_ADDR")
	if addr == "" {
		addr = "localhost:8080"
	}
	c := newConn(transportWS, addr)
	if err := c.dial(); err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	defer c.Close()

	deadline := time.After(4 * time.Second)
	for {
		select {
		case ev := <-c.events:
			chunk, ok := ev.(chunkMsg)
			if !ok {
				continue
			}
			if r := classify(chunk.raw); r.identity != "" {
				t.Logf("IDENTITY received: %s", r.identity)
				return
			}
		case <-deadline:
			t.Errorf("expected an IDENTITY| frame within 4s (server needs DMUD_PERSISTENCE=file)")
			return
		}
	}
}

// TestLiveResumeSmoke proves a character is maintained across connections: get
// the id, rename the character, reconnect with `login <id>`, and confirm the
// resumed character keeps the name. Needs a persistence-enabled server.
func TestLiveResumeSmoke(t *testing.T) {
	if os.Getenv("DMUD_SMOKE") != "1" {
		t.Skip("set DMUD_SMOKE=1 to run the live resume smoke test")
	}
	addr := os.Getenv("DMUD_ADDR")
	if addr == "" {
		addr = "localhost:8080"
	}
	name := fmt.Sprintf("ResumeProbe%d", time.Now().UnixNano()%100000)

	// Connection 1: capture the id, rename, save.
	c1 := newConn(transportWS, addr)
	if err := c1.dial(); err != nil {
		t.Fatalf("dial1: %v", err)
	}
	id := ""
	for d := time.After(3 * time.Second); id == ""; {
		select {
		case ev := <-c1.events:
			if ch, ok := ev.(chunkMsg); ok {
				if r := classify(ch.raw); r.identity != "" {
					id = r.identity
				}
			}
		case <-d:
			c1.Close()
			t.Fatal("no IDENTITY from server (is DMUD_PERSISTENCE=file set?)")
		}
	}
	_ = c1.Send("name " + name)
	time.Sleep(400 * time.Millisecond)
	_ = c1.Send("save")
	time.Sleep(400 * time.Millisecond)
	c1.Close()
	t.Logf("saved character %q under id %s", name, id)

	// Connection 2: log in with the id and confirm the character resumes.
	c2 := newConn(transportWS, addr)
	if err := c2.dial(); err != nil {
		t.Fatalf("dial2: %v", err)
	}
	defer c2.Close()
	_ = c2.Send("login " + id)
	time.Sleep(500 * time.Millisecond)
	_ = c2.Send("who")

	for d := time.After(3 * time.Second); ; {
		select {
		case ev := <-c2.events:
			if ch, ok := ev.(chunkMsg); ok && strings.Contains(ch.raw, name) {
				t.Logf("resumed character %q", name)
				return
			}
		case <-d:
			t.Errorf("resumed session did not keep the name %q", name)
			return
		}
	}
}

// TestLiveNewEventsSmoke exercises the room.info, comms, and combat events
// against a live server. room.info (after look) and comms (after say) are
// deterministic and asserted; combat is best-effort — it fires only if the
// spawn room has an NPC to attack, so it is logged rather than required.
//
//	cd ../../tehran && DMUD_PERSISTENCE=file PORT=8099 ./bin/dmud &
//	DMUD_SMOKE=1 DMUD_ADDR=localhost:8099 go test ./cmd/client/ -run TestLiveNewEventsSmoke -v
func TestLiveNewEventsSmoke(t *testing.T) {
	if os.Getenv("DMUD_SMOKE") != "1" {
		t.Skip("set DMUD_SMOKE=1 to run the live new-events smoke test")
	}
	addr := os.Getenv("DMUD_ADDR")
	if addr == "" {
		addr = "localhost:8080"
	}

	c := newConn(transportWS, addr)
	if err := c.dial(); err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	defer c.Close()

	time.Sleep(1 * time.Second)
	_ = c.Send("look")
	time.AfterFunc(700*time.Millisecond, func() { _ = c.Send("say hello-smoke") })

	var gotInfo, gotComms, gotCombat bool
	npc, attacked := "", false
	deadline := time.After(6 * time.Second)
	for {
		select {
		case ev := <-c.events:
			chunk, ok := ev.(chunkMsg)
			if !ok {
				continue
			}
			r := classify(chunk.raw)
			if r.info != nil {
				gotInfo = true
				t.Logf("room.info: name=%q exits=%v", r.info.Title, r.info.Exits)
			}
			if r.comms != nil {
				gotComms = true
				t.Logf("comms: channel=%s self=%v from=%q text=%q",
					r.comms.Channel, r.comms.Self, r.comms.From, r.comms.Text)
			}
			if r.combat != nil {
				gotCombat = true
				t.Logf("combat: target=%q dmg=%d hp=%d/%d killed=%v",
					r.combat.Target, r.combat.Damage, r.combat.TargetHP, r.combat.TargetMax, r.combat.Killed)
			}
			if r.contents != nil && npc == "" && len(r.contents.NPCs) > 0 {
				npc = r.contents.NPCs[0]
			}
			if npc != "" && !attacked {
				attacked = true
				kw := npc
				if f := strings.Fields(npc); len(f) > 0 {
					kw = f[len(f)-1] // last word is usually the targetable keyword
				}
				time.AfterFunc(300*time.Millisecond, func() { _ = c.Send("kill " + kw) })
			}
		case <-deadline:
			t.Logf("results: room.info=%v comms=%v combat=%v (npc=%q attacked=%v)",
				gotInfo, gotComms, gotCombat, npc, attacked)
			if !gotInfo {
				t.Errorf("expected a room.info event after look")
			}
			if !gotComms {
				t.Errorf("expected a comms event after say")
			}
			switch {
			case gotCombat:
				t.Log("combat event observed")
			case attacked:
				t.Logf("attacked %q but no combat event observed (target may not be attackable)", npc)
			default:
				t.Log("no NPC present to attack; combat event not exercised")
			}
			return
		}
	}
}

// TestLiveEnduranceSmoke drives the anti-spam loop against a live server: spam
// `cast heal` to drain endurance until the server refuses ("too exhausted"),
// then `drink draught` (the starter stamina draught) to restore it. Needs a
// persistence-enabled server so fresh characters get the starter kit.
//
//	cd ../../tehran && DMUD_PERSISTENCE=file PORT=8099 ./bin/dmud &
//	DMUD_SMOKE=1 DMUD_ADDR=localhost:8099 go test ./cmd/client/ -run TestLiveEnduranceSmoke -v
func TestLiveEnduranceSmoke(t *testing.T) {
	if os.Getenv("DMUD_SMOKE") != "1" {
		t.Skip("set DMUD_SMOKE=1 to run the live endurance smoke test")
	}
	addr := os.Getenv("DMUD_ADDR")
	if addr == "" {
		addr = "localhost:8080"
	}

	c := newConn(transportWS, addr)
	if err := c.dial(); err != nil {
		t.Fatalf("dial %s: %v", addr, err)
	}
	defer c.Close()

	time.Sleep(1 * time.Second) // deferred spawn into the world
	for i := 0; i < 6; i++ {
		time.AfterFunc(time.Duration(i)*350*time.Millisecond, func() { _ = c.Send("cast heal") })
	}
	time.AfterFunc(2600*time.Millisecond, func() { _ = c.Send("drink draught") })

	maxEP, minEP, epEnd := 0, 1<<30, 0
	gotEP, sawExhausted := false, false
	deadline := time.After(6 * time.Second)
	for {
		select {
		case ev := <-c.events:
			chunk, ok := ev.(chunkMsg)
			if !ok {
				continue
			}
			if strings.Contains(strings.ToLower(chunk.raw), "exhausted") {
				sawExhausted = true
			}
			if r := classify(chunk.raw); r.status != nil && r.status.HasEP {
				gotEP = true
				if r.status.MaxEP > maxEP {
					maxEP = r.status.MaxEP
				}
				if r.status.EP < minEP {
					minEP = r.status.EP
				}
				epEnd = r.status.EP
				t.Logf("endurance %d/%d", r.status.EP, r.status.MaxEP)
			}
		case <-deadline:
			t.Logf("results: maxEP=%d minEP=%d epEnd=%d exhausted=%v", maxEP, minEP, epEnd, sawExhausted)
			if !gotEP {
				t.Fatal("never received an endurance value in char.vitals")
			}
			if minEP >= maxEP {
				t.Errorf("endurance never dropped (min=%d max=%d) — casting should spend it", minEP, maxEP)
			}
			if !sawExhausted {
				t.Error("spamming heal should hit a 'too exhausted' refusal")
			}
			if epEnd < 60 {
				t.Errorf("drinking a draught should restore endurance; ended at %d", epEnd)
			}
			return
		}
	}
}

func truncate(s string, n int) string {
	s = string([]rune(s)) // keep it simple
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

var _ tea.Msg = chunkMsg{}
