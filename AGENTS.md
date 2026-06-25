## Architecture & "Big Picture"
- **Pattern**: Custom Entity Component System (ECS).
  - **Entities**: Pure IDs (`common.EntityID`), strict data/logic separation.
  - **Components**: Data structs in `internal/components`. Identified by name (e.g., "Health", "Player").
  - **Systems**: Logic in `internal/systems`. Implement `ecs.System` interface.
- **Game Engine**:
  - **Loop**: `Game.loop()` in `internal/game/game.go` runs at ~100Hz (10ms tick). It drives `world.Update()`.
  - **State**: `internal/ecs/world.go` holds all state. Thread-safe via `RWMutex`.
- **Networking**:
  - `net.Server` supports both TCP (raw text) and WebSocket (web client).
  - Client state managed in `Game` struct (sessions, keys).

## Developer Workflows
- **Run/Dev**: `make watch` uses `air` for hot-reloading.
- **Build**: `make build` -> `bin/dmud`.
- **Debugging**:
  - **TCP**: `nc localhost 3333` connects directly to the game loop.
  - **Logs**: Structured logging via `zerolog`. Check stdout/stderr.
- **Testing**: Standard `go test ./...`.

## Conventions & Patterns
- **ECS Data Access**:
  - Access components via `ecs.GetTypedComponent[*T](w, entityID, "ComponentName")`.
  - **Locking Order**: Always `entityMutex` (Write/Read) -> `componentMutex` (Write/Read) to avoid deadlocks.
- **Action Pattern**:
  - Use **Transient Components** for actions. Example: `Movement` component is added to request a move, processed by `MovementSystem`, and immediately removed.
- **Commands**:
  - Register in `internal/game/game.go` -> `initCommands`.
  - Handler signature: `func(player *components.Player, args []string, game *Game)`.
- **Persistence**:
  - Abstraction via `persistence.Store`. Implementations: Memory, Redis, Noop.
  - Use `MakePersistenceKey` helper for consistent key naming.

## Integration & Dependencies
- **Frontend**: The `mud-ui` project (React) connects via WebSocket (`:8080`).
- **Data Flow**:
  - **C->S**: Raw text commands (or `/slash` commands).
  - **S->C**: Unstructured text (chat/logs) OR structured JSON (rare, e.g. status updates).
- **Resources**: JSON files in `resources/` (NPCs, Spawns, Areas) loaded at startup.

## Contributing & Commit Policy
- **No AI co-authorship in git history.** Do **not** add `Co-Authored-By` trailers
  for AI assistants, and do **not** add "Generated with"/AI attribution footers to
  commit messages or PR bodies. This applies to every agent and tool.
- **A human is the author and is accountable for the work.** Commits must be
  authored by the human contributor; agents produce changes *on their behalf*.
  Review the diff before committing — you own what you ship.
- Write commit messages in the imperative mood describing the change and why.
