# TODO - DMUD Technical Debt & Issues

## 🔴 Critical Concurrency Issues

### Health Component Thread Safety
- **File**: `components/health.go:28-44`
- **Issue**: `Heal()` and `TakeDamage()` methods not thread-safe
- **Fix**: Add proper locking around health modifications
- **Priority**: High - Race conditions can cause data corruption

### Player Login State Race
- **File**: `game/game.go:507-529` 
- **Issue**: Player state modification without proper coordination between goroutines
- **Fix**: Use atomic operations or additional synchronization
- **Priority**: High - Can cause duplicate player spawns

### Entity Removal Coordination
- **File**: `ecs/world.go:155-176`
- **Issue**: Entity removal while components still being accessed
- **Fix**: Coordinate entity lifecycle with proper reference counting or staged removal
- **Priority**: High - Can cause crashes

### Combat System Component Access
- **File**: `systems/combat.go:327-329`
- **Issue**: Direct health modification without holding component locks
- **Fix**: Use component locking patterns consistently
- **Priority**: High - Combat calculations can be corrupted

## 🟡 Medium Priority Issues

### WebSocket Client State Management
- **File**: `net/ws_client.go:119-137`
- **Issue**: Ping goroutine lacks proper coordination with connection state
- **Fix**: Add context cancellation and better state coordination
- **Priority**: Medium - Can cause connection leaks

### Login Grace Goroutine Cleanup
- **File**: `game/game.go:498-529`
- **Issue**: Goroutine spawned for login grace period cannot be cancelled
- **Fix**: Add context support for proper cleanup
- **Priority**: Medium - Potential goroutine leaks during high traffic

### Area Broadcast Locking
- **File**: `components/area.go:191-203`
- **Issue**: Holds area lock while calling into player code that may access other resources
- **Fix**: Copy player list or use message passing pattern
- **Priority**: Medium - Can cause contention and potential deadlocks

## Original World Locking Follow-ups

- [ ] Ensure `World.RemoveEntity` acquires `entityMutex` then `componentMutex` within a single critical section so entity and component maps stay consistent.
- [ ] Provide a helper (or documented pattern) for callers that need both entity metadata and component data without violating lock ordering—e.g., a combined read method.
- [ ] Audit all call sites that touch both entities and components to verify they respect the global lock order and don't bypass the helper.

## Combat and Commands

- [ ] Restrict `HandleKill` to local targets so players can't start combat across the world via name lookups (see `Game.HandleKill` in `internal/game/combat.go`).
- [ ] Make auto-retaliation reuse an existing `Combat` component instead of replacing it (see `CombatSystem.Update` in `internal/systems/combat.go`), so we stop wiping custom damage ranges, target queues, or buffs.

## Spawn System

- [ ] Honor `SpawnConfig.RespawnTime`/`Spawn.LastSpawn` in `SpawnSystem.processSpawn` so NPCs obey the timing defined in `resources/spawns.json` instead of respawning every 5 seconds purely by chance.

## Area Broadcast Optimization

- [ ] Refactor `Area.Broadcast` (`internal/components/area.go`) so it doesn't hold `PlayersMutex` while sending to clients; snapshot under a read lock and deliver outside the critical section to prevent slow clients from blocking joins/exits.

## 🟢 Performance Optimizations

### Autosave Frequency
- **File**: `game/game.go:119, 607`
- **Issue**: 500ms autosave interval may be too aggressive for production
- **Fix**: Make configurable or increase to 2-5 seconds
- **Priority**: Low - Performance optimization

## 🔒 Security & Robustness

### Connection Rate Limiting
- **File**: `net/server.go` (connection handling)
- **Issue**: No rate limiting on new connections
- **Fix**: Implement connection rate limiting
- **Priority**: Medium - DoS protection