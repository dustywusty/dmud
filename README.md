# dmud

A small multiplayer text MUD in Go — an ECS game **server** plus a family of
**clients** that connect to it over a simple line-based protocol (WebSocket or
raw TCP). They live in one module but are fully decoupled: run the server once,
point any client at it.

▶ **Play the hosted web client:** [dusty.wtf/projects/dmud](https://dusty.wtf/projects/dmud/)

## What's in here

| | Path | |
|---|---|---|
| **Server** | [`cmd/dmud`](cmd/dmud) + [`internal/`](internal) | the game: ECS, a ~100 Hz loop, WebSocket on `:8080` and raw TCP on `:3333` |
| **Terminal client** | [`cmd/client`](cmd/client/README.md) | the full experience — resizable panels, auto-map, tabbed chat, macros, triggers, aliases |
| **Web client** | [`web/`](web/README.md) | a tiny static browser page, no build step — drop it on a site to let visitors play |
| **Browser bridge** | [`cmd/webtty`](cmd/client/README.md#run-it-in-a-browser) | serves the *terminal* client in a browser via a PTY → xterm.js stream (no WASM) |

The server owns all game state; a client is just a view onto it that speaks the
wire protocol. That decoupling is why the same server backs a TUI, a static web
page, and a browser-streamed terminal without knowing which is connected.

## Quick start

**Run the server:**

```bash
make setup        # first time: install tools + download deps
make dev          # hot-reload dev server on :8080 (WebSocket) and :3333 (TCP)
```

**Connect a client** (in another shell, against that server):

```bash
make client       # native terminal client  → ws://localhost:8080
nc localhost 3333 # or raw TCP from any telnet/netcat client
make web          # static browser client    → http://localhost:8090
make webtty       # the terminal client, in a browser → http://localhost:8091
```

New here? Connect and you'll arrive as a formless spirit — pick a name, race, and
class to manifest into the world. See the per-client READMEs for keys, commands,
and power features: [terminal](cmd/client/README.md) · [web](web/README.md).

## Make targets

Run `make help` for the complete list.

**Server**

| Target | What it does |
|---|---|
| `make dev` | Hot-reload dev server (alias: `make watch`) |
| `make dev-persist` | Dev server with file persistence (saves to `data/`) |
| `make dev-redis` | Start Redis via Compose, then a dev server using it |
| `make build` / `make run` | Compile to `bin/dmud` / build and run |
| `make smoke` | Bot smoke test under the race detector |

**Clients**

| Target | What it does |
|---|---|
| `make client` | Build and run the terminal client (`ARGS="-tcp"` for raw TCP) |
| `make client-build` | Just compile the terminal client to `bin/dmud-client` |
| `make web` | Serve the static web client at `http://localhost:8090` |
| `make webtty` | Run the terminal client in a browser (PTY bridge) at `http://localhost:8091` |

**Project**

| Target | What it does |
|---|---|
| `make test` / `make test-race` | Run all tests / with the race detector |
| `make vet` | `go vet ./...` |
| `make clean` | Remove build artifacts |
| `make docker-build` · `docker-run` · `docker-stop` · `docker-clean` | Docker image lifecycle |
| `make dc-up` · `dc-down` · `dc-logs` | Docker Compose lifecycle |

## Configuration

See [`.env.example`](.env.example) for everything. The common ones:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | WebSocket port (raw TCP is always `:3333`) |
| `DMUD_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `DMUD_PERSISTENCE` | `none` | Persistence driver: `none`, `memory`, `file`, or `redis` |
| `DMUD_DATA_DIR` | `data/` | Where the `file` driver stores characters |
| `DMUD_REDIS_URL` | `redis://localhost:6379/0` | Connection URL for the `redis` driver |

## Docker

```bash
make dc-up                                                   # run the production image
DMUD_PERSISTENCE=redis docker compose --profile redis up --build   # with Redis persistence
```

## Architecture

An Entity Component System (entities are IDs, components are data, systems are
logic) driven by a ~100 Hz game loop. Clients connect over a line-based protocol;
WebSocket clients additionally receive structured `EVENT|…` frames (room contents,
vitals, combat, comms) that power the rich terminal UI, while plain TCP clients get
the same game as readable text. Full deep-dive in [AGENTS.md](AGENTS.md).
