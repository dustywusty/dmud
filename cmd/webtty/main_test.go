package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func newBridge() *bridge {
	return &bridge{perIP: map[string]int{}, maxTotal: 64, maxPerIP: 4}
}

func cookie(v string) *http.Cookie { return &http.Cookie{Name: sessionCookie, Value: v} }

func TestCheckOrigin(t *testing.T) {
	// Empty allow-list = dev mode, allow anything (even no Origin).
	open := newBridge()
	if !open.checkOrigin(httptest.NewRequest("GET", "/ws", nil)) {
		t.Error("empty allow-list should permit any origin")
	}

	b := newBridge()
	b.origins = []string{"https://play.example.com", "localhost:8091"}
	cases := []struct {
		origin string
		want   bool
	}{
		{"https://play.example.com", true}, // exact match
		{"https://play.example.com:443", false},
		{"http://localhost:8091", true}, // host match
		{"https://evil.example.com", false},
		{"", false}, // missing Origin is rejected once a list is set
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", "/ws", nil)
		if c.origin != "" {
			r.Header.Set("Origin", c.origin)
		}
		if got := b.checkOrigin(r); got != c.want {
			t.Errorf("checkOrigin(%q) = %v, want %v", c.origin, got, c.want)
		}
	}
}

func TestAcquireRelease(t *testing.T) {
	b := newBridge()
	b.maxTotal = 3
	b.maxPerIP = 2

	if !b.acquire("1.1.1.1") || !b.acquire("1.1.1.1") {
		t.Fatal("first two from an IP should be admitted")
	}
	if b.acquire("1.1.1.1") {
		t.Error("third from the same IP should be rejected (per-IP cap)")
	}
	if !b.acquire("2.2.2.2") {
		t.Error("a different IP should still be admitted under the per-IP cap")
	}
	// total is now 3 (max). A fourth IP must be rejected by the global cap.
	if b.acquire("3.3.3.3") {
		t.Error("global cap should reject beyond -max")
	}
	// Releasing frees a slot for both caps.
	b.release("1.1.1.1")
	if !b.acquire("1.1.1.1") {
		t.Error("release should free a slot")
	}
}

func TestConfigDirPersistentByCookie(t *testing.T) {
	dir := t.TempDir()
	b := newBridge()
	b.dataDir = dir

	r := httptest.NewRequest("GET", "/ws", nil)
	r.AddCookie(cookie("deadbeefcafef00d"))
	d1, cleanup1, err := b.configDir(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(d1, dir) {
		t.Errorf("persistent dir %q not under data dir %q", d1, dir)
	}
	if _, err := os.Stat(d1); err != nil {
		t.Errorf("persistent dir should exist: %v", err)
	}
	// Write a marker, then "reconnect" with the same cookie: same dir, kept.
	marker := d1 + "/marker"
	if err := os.WriteFile(marker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	cleanup1() // must be a no-op for persistent dirs
	if _, err := os.Stat(marker); err != nil {
		t.Errorf("persistent dir was wiped by cleanup: %v", err)
	}

	r2 := httptest.NewRequest("GET", "/ws", nil)
	r2.AddCookie(cookie("deadbeefcafef00d"))
	d2, _, _ := b.configDir(r2)
	if d2 != d1 {
		t.Errorf("same cookie should map to same dir: %q != %q", d1, d2)
	}
}

func TestConfigDirEphemeralWithoutCookie(t *testing.T) {
	b := newBridge()
	b.dataDir = t.TempDir()

	// No cookie -> temp dir, removed by cleanup even though -data is set.
	r := httptest.NewRequest("GET", "/ws", nil)
	d, cleanup, err := b.configDir(r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(d, b.dataDir) {
		t.Errorf("cookieless session should not land in the persistent data dir")
	}
	cleanup()
	if _, err := os.Stat(d); !os.IsNotExist(err) {
		t.Errorf("ephemeral dir should be removed by cleanup, got %v", err)
	}

	// A bogus (non-hex) cookie value is rejected and treated as ephemeral.
	r2 := httptest.NewRequest("GET", "/ws", nil)
	r2.AddCookie(cookie("../../etc"))
	d2, cleanup2, _ := b.configDir(r2)
	defer cleanup2()
	if strings.HasPrefix(d2, b.dataDir) {
		t.Errorf("malformed token must not be used as a path under data dir: %q", d2)
	}
}

func TestPruneSessions(t *testing.T) {
	dir := t.TempDir()
	b := newBridge()
	b.dataDir = dir
	b.dataTTL = time.Hour

	stale := dir + "/staaaaaaaaaaaaale1"
	fresh := dir + "/freeeeeeeeeeeeesh1"
	for _, d := range []string{stale, fresh} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}

	b.pruneSessions()

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale session dir should have been pruned, got %v", err)
	}
	if _, err := os.Stat(fresh); err != nil {
		t.Errorf("fresh session dir should survive, got %v", err)
	}

	// TTL 0 disables pruning.
	b.dataTTL = 0
	if err := os.Chtimes(fresh, old, old); err != nil {
		t.Fatal(err)
	}
	b.pruneSessions()
	if _, err := os.Stat(fresh); err != nil {
		t.Errorf("with TTL=0 nothing should be pruned, got %v", err)
	}
}

func TestClientIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/ws", nil)
	r.RemoteAddr = "9.9.9.9:5555"
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")

	if got := (&bridge{}).clientIP(r); got != "9.9.9.9" {
		t.Errorf("without trust-proxy, want RemoteAddr host 9.9.9.9, got %q", got)
	}
	if got := (&bridge{trustProxy: true}).clientIP(r); got != "1.2.3.4" {
		t.Errorf("with trust-proxy, want first XFF hop 1.2.3.4, got %q", got)
	}
}

func TestIdleTimerResets(t *testing.T) {
	// Sanity: a reset timer doesn't fire if reset faster than its interval.
	fired := make(chan struct{}, 1)
	tm := time.AfterFunc(40*time.Millisecond, func() { fired <- struct{}{} })
	defer tm.Stop()
	for i := 0; i < 5; i++ {
		time.Sleep(15 * time.Millisecond)
		tm.Reset(40 * time.Millisecond)
	}
	select {
	case <-fired:
		t.Error("idle timer fired despite being reset within its interval")
	default:
	}
}
