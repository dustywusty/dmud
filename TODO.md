# TODO

## World locking follow-ups

- [ ] Ensure `World.RemoveEntity` acquires `entityMutex` then `componentMutex` within a single critical section so entity and component maps stay consistent.
- [ ] Provide a helper (or documented pattern) for callers that need both entity metadata and component data without violating lock ordering—e.g., a combined read method.
- [ ] Audit all call sites that touch both entities and components to verify they respect the global lock order and don’t bypass the helper.

## Combat and commands

- [ ] Restrict `HandleKill` to local targets so players can’t start combat across the world via name lookups (see `Game.HandleKill` in `internal/game/combat.go`).
- [ ] Make auto-retaliation reuse an existing `Combat` component instead of replacing it (see `CombatSystem.Update` in `internal/systems/combat.go`), so we stop wiping custom damage ranges, target queues, or buffs.

## Spawn system

- [ ] Honor `SpawnConfig.RespawnTime`/`Spawn.LastSpawn` in `SpawnSystem.processSpawn` so NPCs obey the timing defined in `resources/spawns.json` instead of respawning every 5 seconds purely by chance.

## Area broadcasts

- [ ] Refactor `Area.Broadcast` (`internal/components/area.go`) so it doesn’t hold `PlayersMutex` while sending to clients; snapshot under a read lock and deliver outside the critical section to prevent slow clients from blocking joins/exits.
