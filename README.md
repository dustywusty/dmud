# dmud

A MUD game server written in Go using an ECS architecture.

Client: [https://dusty.wtf/projects/dmud/](https://dusty.wtf/projects/dmud/)

## Quick Start

```bash
# First-time setup (installs tools, downloads deps)
make setup

# Start dev server with hot reload
make dev

# Run tests
make test
```

## All Targets

Run `make help` to see the full list, or:

| Target | What it does |
|---|---|
| `make setup` | Install tools and dependencies (first-time) |
| `make dev` | Start dev server with hot reload |
| `make build` | Compile binary to `bin/dmud` |
| `make run` | Build and run locally |
| `make test` | Run all tests |
| `make test-race` | Run tests with race detector |
| `make vet` | Run `go vet` |
| `make race` | Run binary with race detector |
| `make clean` | Remove build artifacts |
| `make docker-build` | Build Docker image |
| `make docker-run` | Build and run in Docker (port 8080) |
| `make docker-stop` | Stop running dmud containers |
| `make docker-clean` | Remove dmud Docker image |
| `make dc-up` | Start via Docker Compose |
| `make dc-down` | Stop Docker Compose services |
| `make dc-logs` | Tail Docker Compose logs |

## Docker

```bash
# Run production image locally
make dc-up

# Run with Redis persistence
DMUD_PERSISTENCE=redis docker compose --profile redis up --build
```

## Environment Variables

See [`.env.example`](.env.example) for all options. Key ones:

| Variable | Default | Description |
|---|---|---|
| `DMUD_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |
| `PORT` | `8080` | WebSocket server port |
| `DMUD_PERSISTENCE` | `none` | Persistence driver (none, redis) |
| `DMUD_REDIS_URL` | `redis://...localhost:6379/0` | Redis connection URL |

## Architecture

ECS (Entity Component System) pattern with a ~100Hz game loop. See [AGENT.md](AGENT.md) for the full architecture deep-dive.
