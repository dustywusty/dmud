# dmud-client

A terminal MUD client for the dmud server, built with
[Bubble Tea](https://github.com/charmbracelet/bubbletea) +
[Lip Gloss](https://github.com/charmbracelet/lipgloss). It speaks the server's
existing wire protocol and arranges the stream into adjustable panels.

```
 dmud · WebSocket localhost:8080 · ● connected · MUSHROOM GROVE   Tab focus · ^N/P chat · ^O resize · ^C quit
╭────────────────────────────────────────────────────────────────────────────╮
│ HP ███████████░ 566/600  │  XP ███████░░░ 75/100  │  Lv 1  │  FX bless       │
╰────────────────────────────────────────────────────────────────────────────╯
╭──────────────┐╭────────────────────────────────────┐╭──────────────────────╮
│HERE          ││OUTPUT                              ││MUSHROOM GROVE        │
│◆ Gandalf     ││You stand at a misty crossroads…    ││exits: west, north    │
│· a chicken   ││a sneaky goblin is here.            ││  [◊]───[@]           │
│† a corpse    ││Exits: [west, north]                ││   │                  │
│              ││a goblin attacked you for 13 damage!│╰──────────────────────╯
│              ││You gained 75 experience!           │╭──────────────────────╮
│              ││                                    ││Local · Shout• · Wylie•│
│              ││                                    ││03:58 Gandalf says:   │
│              ││                                    ││  careful, they bite  │
╰──────────────┘╰────────────────────────────────────┘╰──────────────────────╯
╭────────────────────────────────────────────────────────────────────────────╮
│ » look_                                                                      │
╰────────────────────────────────────────────────────────────────────────────╯
```

Panels: a **status bar** (HP/EN/XP/level/effects) across the top — `EN` is the
endurance gauge that abilities spend and that regenerates over time — then **HERE**
(room occupants), **OUTPUT** (main log), **MAP** (current room name + exits +
auto-map), **COMMS** (tabbed chat), and the command input.

- **HERE** lists who and what is in the room: online players `◆` (bright),
  corpses `†` (dim), everything else (NPCs, items) `·`. On a server that sends
  `room.contents` events it updates **live** as people/NPCs come and go; older
  servers fall back to refreshing on `look`. It scrolls (focus it with `Tab`,
  then `PgUp`/`PgDn` or the wheel) when a room is too crowded to fit.
- **COMMS** is tabbed by channel: `Local` (say) and `Shout` always exist, and a
  tab is created per person for private messages (`tell`/`whisper`). `•` marks a
  tab with unread messages; cycle tabs with `^N`/`^P`.
- **MAP** heads with the current room's name and exits, then draws rooms as
  boxes — `[@]` is you, `[ ]` a known room, `[◊]`/`[◈]`
  a room with an up/down exit (stairwell) — linked by `───`/`│`. A connector is
  only drawn when both rooms agree on the exit (no phantom passages), and your
  current room's *unexplored* exits show as green stub ticks (`╶ ╴ ╵ ╷`) — a
  compass embedded in the map. Saves per server, reloads on connect; `/map`
  opens it full-screen, `/map reset` clears it.

## Run

```bash
make client                      # WebSocket -> ws://localhost:8080/ws
make client ARGS="-tcp"          # raw TCP   -> localhost:3333
make client-build                # just compile to bin/dmud-client

# or directly:
go run ./cmd/client -addr myhost:8080
go run ./cmd/client -tcp -addr myhost:3333
go run ./cmd/client -login <uuid>      # restore a saved character on connect
go run ./cmd/client -name Gandalf      # set your name on connect
```

WebSocket is the default and recommended transport: it is the only one that
receives `STATE|` frames, so the HP/XP/level status bar is populated. Over raw
TCP those frames are stripped server-side, so the status bar stays empty.

## Run it in a browser

You can use this exact client — panels, macros, map and all — from a browser,
**without porting anything to WebAssembly** (the Bubble Tea TUI can't compile to
`js/wasm`: no TTY, no terminal). Instead, `cmd/webtty` runs the real binary
*server-side* inside a pseudo-terminal and streams it to
[xterm.js](https://xtermjs.org/) over a WebSocket — the browser is just a
terminal emulator. Keystrokes and resizes flow back to the PTY; xterm.js answers
the client's startup terminal queries (background color, cursor position) like
any real terminal would.

```bash
make webtty        # builds the client + bridge, serves http://localhost:8091
                   # (the spawned client connects to a MUD on :8080)
# behind the scenes:
#   go run ./cmd/webtty -listen :8091 -client bin/dmud-client -mud localhost:8080
```

By default each browser connection spawns its own client process in an isolated,
ephemeral config dir, so sessions don't share or persist state.

**Auto-login + remembered settings.** Add `-data <dir>` and the bridge gives each
browser an `HttpOnly` session cookie and a *persistent* per-cookie config dir.
The client already writes its identity (login id), layout, aliases, macros, map
and history there — so a returning browser **auto-resumes the same character with
the same settings**, no account system needed. The cookie rides the WebSocket
handshake automatically; nothing is bridged through `localStorage` (and `HttpOnly`
means page scripts can't read it). Clearing cookies = a fresh character.

```bash
go run ./cmd/webtty -listen :8091 -mud localhost:8080 \
  -data ./data/webtty            # persistent per-browser identity + settings
```

> **Hosting / security.** The bridge spawns a process per connection, so it's
> heavier per visitor than a normal web app and is a DoS surface. The defaults are
> dev-friendly (any `Origin`, ephemeral); before exposing it publicly, run it
> behind a TLS reverse proxy (`wss://`) and set:
>
> | Flag | Purpose |
> |---|---|
> | `-origins https://you.example` | allow-list browser Origins (blocks cross-site hijacking) |
> | `-max` / `-max-per-ip` | cap concurrent sessions globally / per IP (default 64 / 4) |
> | `-idle 30m` | reap sessions with no input |
> | `-secure-cookie` | set the cookie `Secure` flag (behind TLS) |
> | `-trust-proxy` | read the client IP from `X-Forwarded-For` |
>
> That's enough to host it for a known/community audience. For a fully
> **unauthenticated public** entry point, prefer the lightweight
> [`web/`](../../web) client — it spawns no process and talks straight to the
> MUD's own (origin-checked) WebSocket.

## Keys

| Key            | Action                                          |
|----------------|-------------------------------------------------|
| `Enter`        | send the typed command                          |
| `Shift+1`…`0`  | fire macro 1–10 (on an empty input line); `F1`–`F12` also fire, even mid-typing — see **Macros** |
| `Tab` / `Shift+Tab` | complete the last word; or (empty input) cycle focus through all panes — HERE / OUTPUT / MAP / COMMS |
| `↑` / `↓`      | browse command history (persists across sessions; your draft is kept) |
| `^G`           | toggle the full-screen map                       |
| `^A`           | RULES panel — view/edit macros, aliases, highlights, triggers (`↑↓` select · `e` edit · `d` delete) |
| `^N` / `^P`    | next / previous COMMS channel tab               |
| `^O`           | **resize mode** — `Tab` picks the pane, `←/→` `↑/↓` resize it, `Esc` to finish |
| `Alt+←↑↓→`     | walk west / north / south / east                |
| `PgUp`/`PgDn`  | scroll the focused panel (mouse wheel too)      |
| `^T`           | toggle the whole right column (MAP + COMMS)     |
| `^L`           | clear the OUTPUT panel                          |
| `^C`           | quit                                            |

Resize uses `^O` + plain arrows because modified arrows (`ctrl/alt+arrow`) aren't
delivered reliably across terminals (`ctrl+arrow` is also bound directly for
terminals that do send it). Movement maps both `alt+arrow` and the macOS
word-jump sequences (`alt+b`/`alt+f` for west/east), so Option+arrows work in
Terminal.app and iTerm2. If a binding seems dead, run `/keys` to see exactly what
your terminal sends.

**Speedwalk:** type a run of directions like `3n2e` to send `n n n e e`.

**Macros:** bind a command to a hotkey and a legend strip appears above the
input (`⇧1 Heal · ⇧2 kill rat · …`), shown only while you have macros set.

```
/macro 1 look                       # slot 1 → look
/macro 2 Heal = cast heal on self   # slot 2, labelled "Heal" on the bar
/macro 3 get all; wield sword       # sequences work (so do aliases + speedwalk)
/macro 1                            # clear slot 1
/macro                              # list them
/macro edit 4                       # open the multi-line editor for slot 4
```

**Multi-line / the editor.** A macro body can be **many lines** — one command per
line (each line still supports `;`, aliases, and speedwalk). Open a real editor
with `/macro edit <n>`, or press `^A` (RULES) → select a macro → `e`:

```
MACRO ⇧4   ^S save · Esc cancel · Tab switch field · Enter = new line in body
  label    Pull
▸ commands (one per line; ; and aliases also work)
┃   1 cast heal on self
┃   2 quaff potion
┃   3 say patched up
```

`Enter` adds a line, `Tab` switches between the label and body, `^S` saves, `Esc`
cancels (an empty body clears the slot). Firing runs each line in order; the bar
shows the label (or just the first line) so multi-line macros stay tidy.

**Firing them:** press **`Shift`+a number** (`Shift+1`…`Shift+0` → slots 1–10) on
an **empty input line**. Terminals can't report "Shift+1" separately from the
symbol it types (`!@#$%^&*()`), so the rule is: on an empty line those symbols
fire the bound macro; while you're typing they're ordinary characters — so you
never lose `!` in chat. **`F1`–`F12` also fire macros** (and work mid-typing, plus
reach slots 11–12); on a Mac laptop they may need `fn`. Slots 1–12 are referred to
by number in `/macro` (`1` or `f1`, either parses). Shift+digit assumes a US
keyboard layout; `/keys` shows what your terminal actually sends. Macros live in
`config.json` and also appear in the `^A` RULES overlay to edit or delete.

**Mouse:** the wheel scrolls whichever panel the cursor is over; left-click a
panel to focus it for scrolling, or click a COMMS tab to switch channels. Mouse
capture means the terminal's own text selection is off while it's on — run
`/mouse` to toggle capture when you want to select/copy text.

## Commands & power features

Type `/help` in the client for the full list. Local `/slash` commands never hit
the server:

| Command | What it does |
|---|---|
| `/alias gc get all from corpse` | define an alias (`;` chains commands; `/alias gc` removes it) |
| `/highlight yellow tells you`   | colorize output matching a regex (`/highlight off` clears) |
| `/trigger low health = quaff red = /bell` | auto-run a command (or `/bell`) when output matches |
| `/map` · `/map reset`           | full-screen map · wipe the saved map and start fresh |
| `/spacing`                      | toggle blank lines between messages (roomy ↔ compact) |
| `/mouse`                        | toggle mouse capture (off to select/copy text) |
| `/find <text>`                  | jump OUTPUT to the latest match |
| `/log [path]`                   | start/stop writing the session to a file |
| `/reconnect`                    | drop and redial |
| `/clear`, `/quit`, `/bell`      | clear output · leave · ring the bell |

- **Auto-reconnect:** a dropped connection retries with backoff (1s→30s) and
  replays your `login`/`name`; typing `exit` or `/quit` suppresses it.
- **Persistence:** layout, aliases, highlights, and triggers live in
  `$XDG_CONFIG_HOME/dmud-client/config.json` (default `~/.config/dmud-client/`);
  the explored map is saved per server under `maps/<host>.json` and reloaded on
  connect; command history is kept in `history` (most recent 1000, `login` lines
  excluded).
- **Auto-resume:** when the server runs with persistence on, it hands the client
  a login id (`IDENTITY|…`). The client stores it per server in `identity.json`
  and replays `login <id>` on connect, so you come back as the same character —
  no `save`/copy-paste. (`-login <uuid>` still overrides it.)

## How it routes the stream

The server sends line-based text. Some frames are structured/tagged. The client
(`protocol.go`) classifies each chunk:

| Server output                          | Panel                          |
|----------------------------------------|--------------------------------|
| `EVENT\|{json}` `room.contents`         | HERE (live who/what is here)   |
| `EVENT\|{json}` `room.info`             | MAP/ROOM name + exits (authoritative) |
| `EVENT\|{json}` `char.vitals`           | status bar (HP/XP/level/effects) |
| `EVENT\|{json}` `comms`                 | COMMS (routed by channel)      |
| `EVENT\|{json}` `combat`                | enemy HP bar (status bar)      |
| `STATE\|HP:..\|LEVEL:..\|XP:..\|AREA:..` | status bar (legacy fallback)   |
| `STATUS\|…`                             | OUTPUT (notice, prefix stripped) |
| `DMG\|…` / `DMG\|DEATH\|…`               | OUTPUT (combat, prefix stripped) |
| `CHAT\|…`                               | COMMS                          |
| a block containing `Exits: [..]`        | ROOM + MAP (+ body to OUTPUT)  |
| `X says:` / `X shouts:` / `You say:`    | COMMS                          |
| everything else                         | OUTPUT                         |

The MAP is built from the direction of each movement command plus the room
title in the block that follows, so it draws edges as you walk.

**Structured events (`EVENT|{json}`)** are an additive, GMCP-style side-channel
the server pushes (WS only — `SupportsTags`) alongside the text. The client
decodes:

- `room.contents` — live HERE as players/NPCs/items come and go.
- `room.info` — authoritative room name + exits driving the MAP/header (the
  `Exits: [..]` text scrape is kept as a fallback for transports without events).
- `char.vitals` — HP/EN(endurance)/XP/level/effects → status bar.
- `comms` — chat routed to a COMMS tab by its `channel` (say→Local, shout→Shout,
  tell→per-person). Tag clients get this *instead of* the plain "X says: …" line,
  so there's no duplication; plain-text/web clients still get the legacy text.
- `combat` — the attack target's name + current/max HP, rendered as an enemy bar
  in the status line that drains as you fight and reads "slain" on a kill (it
  clears when you change rooms).

The text narrative is unchanged. Unknown event types are ignored, so the server
can add more packages without breaking older clients. See `parseEvent` in
`protocol.go`.

Classification reuses the server's own `internal/util` helpers (`StripTag`,
`IsStateMessage`) so the two never drift. The chat/room heuristics are isolated
in `classify()`; if the server starts emitting real `CHAT|`/`DMG|`/`SYS|` tags
(the `util.TagMessage` machinery already exists), routing can become exact with
a one-line change.

## Files

- `main.go` — flags, program bootstrap
- `conn.go` — WebSocket + TCP transports, reader → event channel
- `protocol.go` — `STATE`/`STATUS`/`DMG`/tag parsing and chunk classification
- `mapper.go` — the auto-mapper: room graph, grid placement, ASCII render
- `model.go` — Bubble Tea model: state, `Update`, layout/resize, reconnect, log
- `commands.go` — input pipeline: aliases, speedwalk, completion, `/slash` cmds
- `rules.go` — compiled highlight/trigger rules
- `view.go` — panel rendering (room, compass, occupants, map, status bars)
- `theme.go` — Lip Gloss palette and the progress-bar helper
- `config.go` — load/save config (layout + aliases + highlights + triggers)

## Tests

```bash
go test ./cmd/client/                                    # headless unit + render tests
DMUD_DUMP=1  go test ./cmd/client/ -run TestDumpLayout -v        # print a layout snapshot
DMUD_SMOKE=1 go test ./cmd/client/ -run TestLiveServerSmoke -v   # against a live server
```
