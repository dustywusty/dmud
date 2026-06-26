# dmud web client

A tiny standalone web client — one `index.html`, no build step, no dependencies.
Command line in, game text out. Drop it on your site so visitors can briefly
play from the browser.

## Point it at your server

The page connects to your dmud server's WebSocket (`/ws`). Pick one:

- **Same host** (the page is served from the same host as the server): leave the
  defaults — it uses `wss://<this-host>/ws`.
- **Hosted elsewhere** (e.g. your personal site, server is on another host): set
  `SERVER` near the top of `index.html`, e.g.
  ```js
  const SERVER = "wss://dmud.example.com/ws";
  ```
- **Per-visit override**: `?ws=wss://host/ws` in the URL.

## Host it

It's a static file — put `index.html` anywhere static (GitHub Pages, Cloudflare
Pages, an S3 bucket, your own site) and link or embed it:

```html
<iframe src="https://yoursite.example/dmud/" style="width:100%;height:600px;border:0"></iframe>
```

**Origin allow-list:** the server only accepts WebSocket upgrades from approved
origins (`internal/net/ws_client.go`, `CheckOrigin`). `dusty.wtf`,
`*.pages.dev`, `*.run.app`, and `localhost` are already allowed — add your site's
hostname there if it differs.

## Spectator mode

Add `?spectate` to the URL for a read-only embed: the command input is hidden,
the bar shows a `👁 spectating` marker, and the page auto-issues a `look` on
connect so there's something to watch immediately. Ideal for a "peek at the live
game" panel on a personal site without inviting strangers to type.

```html
<iframe src="https://yoursite.example/dmud/?spectate"
        style="width:100%;height:600px;border:0"></iframe>
```

Combine with `?ws=` if the server lives elsewhere:
`?spectate&ws=wss://dmud.example.com/ws`.

## Test locally

With a dmud server running on `:8080`:

```bash
make web    # serves web/ at http://localhost:8090
# then open:  http://localhost:8090/?ws=ws://localhost:8080/ws
```

## What it shows

Plain game text. The server's structured frames (`STATE|`, `EVENT|`, `IDENTITY|`)
are hidden; `STATUS|`/`DMG|`/`CHAT|` prefixes are stripped and lightly colored
(system / combat / chat). Includes auto-reconnect and `↑`/`↓` command history.

**Returning visitors:** if the server runs with persistence on, it sends a login
id (`IDENTITY|…`) which the page stashes in `localStorage` and replays as
`login <id>` on the next visit — so a returning browser resumes the same
character.
It is intentionally minimal — for the full experience (map, panels, tabs) use the
terminal client in `cmd/client`. If you want *that* full client in a browser,
`cmd/webtty` streams the real TUI to xterm.js over a PTY bridge (`make webtty`);
it's a local/dev tool, not a public entry point — this plain client is the one to
expose to strangers.
