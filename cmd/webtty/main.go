// Command webtty serves the dmud TUI client in a browser. For each connection it
// spawns the real cmd/client binary in a pseudo-terminal and streams it to
// xterm.js over a WebSocket — so the browser shows the exact terminal client,
// with no WASM port. Keystrokes and resizes flow back to the PTY.
//
//	make webtty            # serves http://localhost:8091, client → ws://localhost:8080/ws
//
// # Persistence (auto-login + settings)
//
// With -data set, each browser gets an HttpOnly session cookie and a persistent
// per-cookie config dir (XDG_CONFIG_HOME). The TUI client already writes its
// identity (login id), layout, aliases, macros, map and history there, so a
// returning browser auto-resumes the same character with the same settings — no
// localStorage bridging needed; the cookie rides the WebSocket handshake. Without
// -data, sessions are ephemeral (a temp dir, wiped on disconnect).
//
// # Hosting / security
//
// This spawns a process per connection, so it is heavier per visitor than a normal
// web app and is a DoS surface. Before exposing it publicly:
//   - -origins  : allow-list of browser Origins (CSWSH protection); empty = allow all (dev)
//   - -max / -max-per-ip : cap concurrent sessions globally and per client IP
//   - -idle     : reap sessions with no input after this long
//   - -secure-cookie : set the Secure flag (enable when behind TLS)
//   - run it behind a TLS reverse proxy (wss://) and add -trust-proxy to read X-Forwarded-For
//
// For an unauthenticated public entry point, prefer the static web/ client, which
// spawns no process and talks straight to the MUD's own (origin-checked) WebSocket.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

const sessionCookie = "dmud_sess"

// tokenRE guards values that become a filesystem path under -data. Our own cookie
// is hex, so anything else is treated as no token (ephemeral session).
var tokenRE = regexp.MustCompile(`^[a-f0-9]{8,64}$`)

type bridge struct {
	clientPath   string
	mud          string
	useTCP       bool
	dataDir      string // "" = ephemeral per-connection config dirs
	dataTTL      time.Duration
	idle         time.Duration
	maxTotal     int
	maxPerIP     int
	trustProxy   bool
	secureCookie bool
	origins      []string
	upgrader     websocket.Upgrader

	mu    sync.Mutex
	total int
	perIP map[string]int
}

func main() {
	listen := flag.String("listen", ":8091", "address to serve the browser terminal on")
	clientPath := flag.String("client", "bin/dmud-client", "path to the dmud-client TUI binary")
	mud := flag.String("mud", "localhost:8080", "MUD server address the spawned client connects to")
	useTCP := flag.Bool("tcp", false, "have the spawned client use raw TCP instead of WebSocket")
	dataDir := flag.String("data", "", "persist per-browser config here (auto-login + settings); empty = ephemeral")
	dataTTL := flag.Duration("data-ttl", 30*24*time.Hour, "prune persistent session dirs unused for this long (0 = never)")
	origins := flag.String("origins", "", "comma-separated allowed browser Origins; empty = allow all (dev only)")
	maxTotal := flag.Int("max", 64, "max concurrent sessions (0 = unlimited)")
	maxPerIP := flag.Int("max-per-ip", 4, "max concurrent sessions per client IP (0 = unlimited)")
	idle := flag.Duration("idle", 30*time.Minute, "kill a session after this long with no input (0 = never)")
	trustProxy := flag.Bool("trust-proxy", false, "read client IP from X-Forwarded-For (only behind a trusted proxy)")
	secureCookie := flag.Bool("secure-cookie", false, "set the Secure flag on the session cookie (enable behind TLS)")
	flag.Parse()

	b := &bridge{
		clientPath:   *clientPath,
		mud:          *mud,
		useTCP:       *useTCP,
		dataDir:      *dataDir,
		dataTTL:      *dataTTL,
		idle:         *idle,
		maxTotal:     *maxTotal,
		maxPerIP:     *maxPerIP,
		trustProxy:   *trustProxy,
		secureCookie: *secureCookie,
		perIP:        map[string]int{},
	}
	for _, o := range strings.Split(*origins, ",") {
		if o = strings.TrimSpace(o); o != "" {
			b.origins = append(b.origins, o)
		}
	}
	b.upgrader = websocket.Upgrader{CheckOrigin: b.checkOrigin}

	if b.dataDir != "" {
		if err := os.MkdirAll(b.dataDir, 0o700); err != nil {
			log.Fatalf("data dir: %v", err)
		}
		b.pruneSessions() // sweep stale dirs at startup, then periodically
		go func() {
			for range time.Tick(6 * time.Hour) {
				b.pruneSessions()
			}
		}()
	}

	http.HandleFunc("/", b.serveIndex)
	http.HandleFunc("/ws", b.serveTerm)

	mode := "ephemeral"
	if b.dataDir != "" {
		mode = "persistent (" + b.dataDir + ")"
	}
	origin := "any origin"
	if len(b.origins) > 0 {
		origin = strings.Join(b.origins, ",")
	}
	log.Printf("dmud web terminal: http://%s  client=%s mud=%s  sessions=%s  allow=%s",
		*listen, b.clientPath, b.mud, mode, origin)
	log.Fatal(http.ListenAndServe(*listen, nil))
}

// checkOrigin enforces the Origin allow-list. Empty list = allow all (dev). A
// browser sends Origin on the WebSocket handshake, so this blocks cross-site
// WebSocket hijacking from other pages.
func (b *bridge) checkOrigin(r *http.Request) bool {
	if len(b.origins) == 0 {
		return true
	}
	o := r.Header.Get("Origin")
	if o == "" {
		return false
	}
	host := o
	if u, err := url.Parse(o); err == nil && u.Host != "" {
		host = u.Host
	}
	for _, a := range b.origins {
		if a == o || a == host {
			return true
		}
	}
	return false
}

// serveIndex serves the xterm.js page and, when persistence is on, ensures the
// browser holds a session cookie (set on this normal HTTP response, where
// Set-Cookie works; it then rides the later /ws handshake automatically).
func (b *bridge) serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if b.dataDir != "" {
		if _, err := r.Cookie(sessionCookie); err != nil {
			http.SetCookie(w, &http.Cookie{
				Name:     sessionCookie,
				Value:    newToken(),
				Path:     "/",
				MaxAge:   365 * 24 * 60 * 60,
				HttpOnly: true,
				Secure:   b.secureCookie,
				SameSite: http.SameSiteLaxMode,
			})
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, indexHTML)
}

func (b *bridge) serveTerm(w http.ResponseWriter, r *http.Request) {
	ip := b.clientIP(r)
	if !b.acquire(ip) {
		http.Error(w, "too many sessions, try again shortly", http.StatusServiceUnavailable)
		return
	}
	defer b.release(ip)

	conn, err := b.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	cfgDir, cleanup, err := b.configDir(r)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("setup failed: "+err.Error()))
		return
	}
	defer cleanup()

	args := []string{"-addr", b.mud}
	if b.useTCP {
		args = append(args, "-tcp")
	}
	cmd := exec.Command(b.clientPath, args...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color", "XDG_CONFIG_HOME="+cfgDir)

	// Start with a size so Bubble Tea paints its first frame immediately (it reads
	// the terminal size at startup); the browser's resize frame refines it.
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: 120, Rows: 32})
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("failed to start client: "+err.Error()))
		return
	}
	defer func() {
		_ = ptmx.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	// Reap abandoned sessions: a half-open tab stops sending input but the process
	// would otherwise linger. The timer resets on each keystroke (see below).
	var idleTimer *time.Timer
	if b.idle > 0 {
		idleTimer = time.AfterFunc(b.idle, func() { _ = cmd.Process.Kill() })
		defer idleTimer.Stop()
	}

	// PTY output -> browser (binary frames).
	go func() {
		buf := make([]byte, 8192)
		for {
			n, rerr := ptmx.Read(buf)
			if n > 0 {
				if werr := conn.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					return
				}
			}
			if rerr != nil {
				conn.Close()
				return
			}
		}
	}()

	// Browser -> PTY. A frame beginning with NUL is a resize control message
	// (\x00 + JSON {Cols,Rows}); everything else is keystroke input.
	for {
		_, msg, rerr := conn.ReadMessage()
		if rerr != nil {
			return
		}
		if idleTimer != nil {
			idleTimer.Reset(b.idle)
		}
		if len(msg) > 0 && msg[0] == 0 {
			var rs struct{ Cols, Rows uint16 }
			if json.Unmarshal(msg[1:], &rs) == nil && rs.Cols > 0 && rs.Rows > 0 {
				_ = pty.Setsize(ptmx, &pty.Winsize{Cols: rs.Cols, Rows: rs.Rows})
			}
			continue
		}
		if _, werr := ptmx.Write(msg); werr != nil {
			return
		}
	}
}

// configDir picks the client's XDG_CONFIG_HOME. With -data and a valid session
// cookie it returns a persistent per-browser dir (so identity + settings survive
// across visits); otherwise a temp dir wiped on disconnect.
func (b *bridge) configDir(r *http.Request) (dir string, cleanup func(), err error) {
	if b.dataDir != "" {
		if c, cerr := r.Cookie(sessionCookie); cerr == nil && tokenRE.MatchString(c.Value) {
			dir = filepath.Join(b.dataDir, c.Value)
			if err = os.MkdirAll(dir, 0o700); err != nil {
				return "", nil, err
			}
			now := time.Now()
			_ = os.Chtimes(dir, now, now) // mark last-used for the TTL sweep
			return dir, func() {}, nil     // persistent: keep it
		}
	}
	dir, err = os.MkdirTemp("", "dmud-webtty-")
	if err != nil {
		return "", nil, err
	}
	return dir, func() { _ = os.RemoveAll(dir) }, nil
}

// pruneSessions removes persistent per-cookie config dirs that haven't been
// used within dataTTL (each connection touches its dir's mtime). Keeps disk from
// growing without bound when -data is set.
func (b *bridge) pruneSessions() {
	if b.dataDir == "" || b.dataTTL <= 0 {
		return
	}
	entries, err := os.ReadDir(b.dataDir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-b.dataTTL)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.RemoveAll(filepath.Join(b.dataDir, e.Name()))
		}
	}
}

func (b *bridge) acquire(ip string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.maxTotal > 0 && b.total >= b.maxTotal {
		return false
	}
	if b.maxPerIP > 0 && b.perIP[ip] >= b.maxPerIP {
		return false
	}
	b.total++
	b.perIP[ip]++
	return true
}

func (b *bridge) release(ip string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.total--
	if b.perIP[ip]--; b.perIP[ip] <= 0 {
		delete(b.perIP, ip)
	}
}

func (b *bridge) clientIP(r *http.Request) string {
	if b.trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			return strings.TrimSpace(strings.Split(xff, ",")[0])
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

func newToken() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

const indexHTML = `<!doctype html>
<html><head><meta charset="utf-8"><title>dmud</title>
<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/xterm@5.3.0/css/xterm.css">
<style>html,body{margin:0;height:100%;background:#0d0f12}#t{height:100%;width:100%}</style>
</head><body><div id="t"></div>
<script src="https://cdn.jsdelivr.net/npm/xterm@5.3.0/lib/xterm.js"></script>
<script src="https://cdn.jsdelivr.net/npm/xterm-addon-fit@0.8.0/lib/xterm-addon-fit.js"></script>
<script>
  const term = new Terminal({cursorBlink:true, fontFamily:'ui-monospace,Menlo,Consolas,monospace', fontSize:14, theme:{background:'#0d0f12'}});
  const fit = new FitAddon.FitAddon();
  term.loadAddon(fit);
  term.open(document.getElementById('t'));
  fit.fit();
  const ws = new WebSocket((location.protocol==='https:'?'wss://':'ws://')+location.host+'/ws');
  ws.binaryType = 'arraybuffer';
  function sendResize(){ try { ws.send('\x00'+JSON.stringify({Cols:term.cols,Rows:term.rows})); } catch(e){} }
  ws.onmessage = e => term.write(new Uint8Array(e.data));
  ws.onopen = () => { sendResize(); term.focus(); };
  ws.onclose = () => term.write('\r\n\x1b[31m[disconnected]\x1b[0m\r\n');
  term.onData(d => ws.send(d));
  addEventListener('resize', () => { fit.fit(); sendResize(); });
  term.onResize(sendResize);
</script></body></html>
`
