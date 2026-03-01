# Active Context - dmud

<!-- Updated by both Architect and Executor after each session -->

## Current Focus

Integration test suite is implemented and passing.

## Recent Changes

| Date | Change | Agent |
|------|--------|-------|
| 2026-02-28 | Initialized agentic workspace | Tocket CLI |
| 2026-02-28 | Added `RunWithListener` + `initWSMux` to `internal/net/server.go` | Claude Code |
| 2026-02-28 | Created `test/integration/integration_test.go` (10 tests, all passing) | Claude Code |
| 2026-02-28 | Added `cast charm` spell to `internal/game/misc.go` | Claude Code |

## Open Decisions

_None._

## Recent Changes (continued)

| Date | Change | Agent |
|------|--------|-------|
| 2026-02-28 | Refactored integration tests: added `waitForNPC` helper, `t.Parallel()` on all tests, fixed redundant sleep in TestCastCharm | Claude Code |
| 2026-02-28 | Makefile: added `-timeout 120s` to test targets, added `test-integration` target, removed dead `connect` target | Claude Code |
| 2026-02-28 | README: removed dev artifact (`find internal...`), updated deployment note to Docker | Claude Code |
