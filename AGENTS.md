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

---

<!-- rtk-instructions v2 -->
# RTK (Rust Token Killer) - Token-Optimized Commands

## Golden Rule

**Always prefix commands with `rtk`**. If RTK has a dedicated filter, it uses it. If not, it passes through unchanged. This means RTK is always safe to use.

**Important**: Even in command chains with `&&`, use `rtk`:
```bash
# ❌ Wrong
git add . && git commit -m "msg" && git push

# ✅ Correct
rtk git add . && rtk git commit -m "msg" && rtk git push
```

## RTK Commands by Workflow

### Build & Compile (80-90% savings)
```bash
rtk cargo build         # Cargo build output
rtk cargo check         # Cargo check output
rtk cargo clippy        # Clippy warnings grouped by file (80%)
rtk tsc                 # TypeScript errors grouped by file/code (83%)
rtk lint                # ESLint/Biome violations grouped (84%)
rtk prettier --check    # Files needing format only (70%)
rtk next build          # Next.js build with route metrics (87%)
```

### Test (60-99% savings)
```bash
rtk cargo test          # Cargo test failures only (90%)
rtk go test             # Go test failures only (90%)
rtk jest                # Jest failures only (99.5%)
rtk vitest              # Vitest failures only (99.5%)
rtk playwright test     # Playwright failures only (94%)
rtk pytest              # Python test failures only (90%)
rtk rake test           # Ruby test failures only (90%)
rtk rspec               # RSpec test failures only (60%)
rtk test <cmd>          # Generic test wrapper - failures only
```

### Git (59-80% savings)
```bash
rtk git status          # Compact status
rtk git log             # Compact log (works with all git flags)
rtk git diff            # Compact diff (80%)
rtk git show            # Compact show (80%)
rtk git add             # Ultra-compact confirmations (59%)
rtk git commit          # Ultra-compact confirmations (59%)
rtk git push            # Ultra-compact confirmations
rtk git pull            # Ultra-compact confirmations
rtk git branch          # Compact branch list
rtk git fetch           # Compact fetch
rtk git stash           # Compact stash
rtk git worktree        # Compact worktree
```

Note: Git passthrough works for ALL subcommands, even those not explicitly listed.

### GitHub (26-87% savings)
```bash
rtk gh pr view <num>    # Compact PR view (87%)
rtk gh pr checks        # Compact PR checks (79%)
rtk gh run list         # Compact workflow runs (82%)
rtk gh issue list       # Compact issue list (80%)
rtk gh api              # Compact API responses (26%)
```

### JavaScript/TypeScript Tooling (70-90% savings)
```bash
rtk pnpm list           # Compact dependency tree (70%)
rtk pnpm outdated       # Compact outdated packages (80%)
rtk pnpm install        # Compact install output (90%)
rtk npm run <script>    # Compact npm script output
rtk npx <cmd>           # Compact npx command output
rtk prisma              # Prisma without ASCII art (88%)
```

### Files & Search (60-75% savings)
```bash
rtk ls <path>           # Tree format, compact (65%)
rtk read <file>         # Code reading with filtering (60%)
rtk grep <pattern>      # Search grouped by file (75%). Format flags (-c, -l, -L, -o, -Z) run raw.
rtk find <pattern>      # Find grouped by directory (70%)
```

### Analysis & Debug (70-90% savings)
```bash
rtk err <cmd>           # Filter errors only from any command
rtk log <file>          # Deduplicated logs with counts
rtk json <file>         # JSON structure without values
rtk deps                # Dependency overview
rtk env                 # Environment variables compact
rtk summary <cmd>       # Smart summary of command output
rtk diff                # Ultra-compact diffs
```

### Infrastructure (85% savings)
```bash
rtk docker ps           # Compact container list
rtk docker images       # Compact image list
rtk docker logs <c>     # Deduplicated logs
rtk kubectl get         # Compact resource list
rtk kubectl logs        # Deduplicated pod logs
```

### Network (65-70% savings)
```bash
rtk curl <url>          # Compact HTTP responses (70%)
rtk wget <url>          # Compact download output (65%)
```

### Meta Commands
```bash
rtk gain                # View token savings statistics
rtk gain --history      # View command history with savings
rtk discover            # Analyze Claude Code sessions for missed RTK usage
rtk proxy <cmd>         # Run command without filtering (for debugging)
rtk init                # Add RTK instructions to CLAUDE.md
rtk init --global       # Add RTK to ~/.claude/CLAUDE.md
```

## Token Savings Overview

| Category | Commands | Typical Savings |
|----------|----------|-----------------|
| Tests | vitest, playwright, cargo test | 90-99% |
| Build | next, tsc, lint, prettier | 70-87% |
| Git | status, log, diff, add, commit | 59-80% |
| GitHub | gh pr, gh run, gh issue | 26-87% |
| Package Managers | pnpm, npm, npx | 70-90% |
| Files | ls, read, grep, find | 60-75% |
| Infrastructure | docker, kubectl | 85% |
| Network | curl, wget | 65-70% |

Overall average: **60-90% token reduction** on common development operations.
<!-- /rtk-instructions -->
