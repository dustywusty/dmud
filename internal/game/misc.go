package game

import (
	"context"
	"dmud/internal/common"
	"dmud/internal/components"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jedib0t/go-pretty/table"
	"github.com/rs/zerolog/log"
)

func handleLook(player *components.Player, args []string, game *Game) {
	// "look <target>" examines that target; bare "look" surveys the area.
	if len(args) > 0 {
		handleExamine(player, args, game)
		return
	}
	player.Look(game.world.AsWorldLike())
}

func handleWho(player *components.Player, args []string, game *Game) {
	game.playersMu.Lock()
	defer game.playersMu.Unlock()

	tw := table.NewWriter()
	tw.SetStyle(table.StyleLight)
	tw.AppendHeader(table.Row{"Player", "Race", "Level", "Online Since"})

	for _, playerEntity := range game.players {
		playerComponent, err := game.world.GetComponent(playerEntity.ID, "Player")
		if err != nil {
			log.Error().Err(err).Msgf("Could not get component for player %s", playerEntity.ID)
			continue
		}
		playerData, ok := playerComponent.(*components.Player)
		if !ok {
			log.Error().Msgf("Error type asserting component for player %s", playerEntity.ID)
			continue
		}

		// Get player level
		level := 1
		expComponent, err := game.world.GetComponent(playerEntity.ID, "Experience")
		if err == nil {
			if exp, ok := expComponent.(*components.Experience); ok {
				level = exp.GetLevel()
			}
		}

		tw.AppendRow(table.Row{playerData.Name, "??", level, playerEntity.CreatedAt.DiffForHumans()})
	}

	player.Broadcast(tw.Render())
}

func handleExit(player *components.Player, args []string, game *Game) {
	player.RWMutex.RLock()
	defer player.RWMutex.RUnlock()

	game.HandleDisconnect(player.Client)
}

func handleName(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Usage: name <new_name>")
		return
	}
	newName := args[0]
	game.HandleRename(player, newName)
}

func handleLogin(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Usage: login <uuid>")
		return
	}
	raw := strings.TrimSpace(args[0])
	parsed, err := uuid.Parse(raw)
	if err != nil || parsed.Version() != 4 {
		player.Broadcast("Invalid UUID. Expected a UUIDv4.")
		return
	}
	game.HandleLogin(player, parsed.String())
}

func handleSave(player *components.Player, args []string, game *Game) {
	game.HandleSave(player)
}

func handleCast(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Usage: cast <spell> [target]")
		player.Broadcast("Known spells: " + strings.Join(listKnownSpells(), ", "))
		player.Broadcast("Type 'spells' for your spellbook with level requirements.")
		return
	}

	spell, consumed := resolveSpellFromArgs(args)
	if spell == nil {
		player.Broadcast("You don't know that spell.")
		player.Broadcast("Known spells: " + strings.Join(listKnownSpells(), ", "))
		return
	}

	casterID, idErr := game.getPlayerEntity(player)

	// Beast forms can't speak incantations — shift is melee, casting is humanoid.
	if idErr == nil && game.playerShift(casterID) != nil {
		player.Broadcast("You can't weave spells in beast form -- 'revert' to your own shape first.")
		return
	}

	// Class discipline: a chosen class can only cast its own schools (classless
	// casters are unrestricted).
	if idErr == nil && !classAllowsSchool(game.getStats(casterID), spell.School) {
		player.Broadcast(fmt.Sprintf("Your discipline doesn't include %s magic.", spell.School))
		return
	}

	// Level gate: higher-tier spells require training (a class level).
	if idErr == nil && spell.MinLevel > 1 {
		if lvl := game.casterLevel(casterID); lvl < spell.MinLevel {
			player.Broadcast(fmt.Sprintf("You aren't skilled enough to cast %s yet -- it requires level %d (you are level %d).",
				spell.Name, spell.MinLevel, lvl))
			return
		}
	}

	// Endurance gate: spells cost endurance, so they can't be spammed. Dexterity
	// makes casting more efficient (lower effective cost). Refuse the cast
	// (changing nothing) when the caster is too exhausted.
	effCost := spell.Cost
	if effCost > 0 && idErr == nil {
		if stats := game.getStats(casterID); stats != nil {
			if effCost = int(float64(effCost) * stats.CostFactor()); effCost < 1 {
				effCost = 1
			}
		}
		if end := getEndurance(game, casterID); end != nil {
			if !end.Spend(effCost) {
				end.Regen()
				player.Broadcast(fmt.Sprintf("You're too exhausted to cast %s — it needs %d endurance and you have %d/%d.",
					spell.Name, effCost, end.Current, end.Max))
				player.BroadcastState(game.world.AsWorldLike(), casterID)
				return
			}
		}
	}

	spell.Handler(player, args[consumed:], game)

	// Refresh the caster's vitals so the endurance bar reflects the spend.
	if spell.Cost > 0 && idErr == nil {
		player.BroadcastState(game.world.AsWorldLike(), casterID)
	}
}

// getStats returns a player entity's Stats component, or nil if absent.
func (g *Game) getStats(entityID common.EntityID) *components.Stats {
	if comp, err := g.world.GetComponent(entityID, "Stats"); err == nil {
		if s, ok := comp.(*components.Stats); ok {
			return s
		}
	}
	return nil
}

// casterLevel returns a player entity's character level (1 if unknown).
func (g *Game) casterLevel(entityID common.EntityID) int {
	if exp, err := g.world.GetComponent(entityID, "Experience"); err == nil {
		if e, ok := exp.(*components.Experience); ok {
			return e.GetLevel()
		}
	}
	return 1
}

// getEndurance returns the entity's Endurance component, or nil if absent.
func getEndurance(game *Game, entityID common.EntityID) *components.Endurance {
	comp, err := game.world.GetComponent(entityID, "Endurance")
	if err != nil {
		return nil
	}
	end, _ := comp.(*components.Endurance)
	return end
}

// handleRace shows or changes the player's race. Changing it reforges stats to
// that race's baseline (and updates size), so it's a fresh start as that race.
func (g *Game) handleRace(player *components.Player, args []string, game *Game) {
	entityID, err := g.getPlayerEntity(player)
	if err != nil {
		player.Broadcast("Unable to read your character.")
		return
	}
	stats := g.getStats(entityID)
	if stats == nil {
		player.Broadcast("You have no stats.")
		return
	}

	if len(args) == 0 {
		player.Broadcast(fmt.Sprintf("You are a %s %s.", stats.Size, stats.Race))
		player.Broadcast("Races: " + strings.Join(components.RaceNames(), ", ") + "   (race <name> reforges your stats)")
		return
	}

	key := strings.ToLower(strings.TrimSpace(args[0]))
	def, ok := components.RaceFor(key)
	if !ok {
		player.Broadcast("No such race. Try: " + strings.Join(components.RaceNames(), ", "))
		return
	}
	// Reforging to a race rebases stats — preserve any chosen class (and its
	// affinity) so race/class can be picked in any order during creation.
	prevClass := stats.Class
	*stats = *components.NewStatsForRace(key)
	if prevClass != "" {
		stats.Class = prevClass
		if cdef, ok := classByKey(prevClass); ok {
			stats.Set(cdef.PrimaryStat, stats.Get(cdef.PrimaryStat)+classAffinityBonus)
		}
	}
	player.Broadcast(fmt.Sprintf("You are reforged as a %s %s.", def.Size, def.Name))
	g.handleScore(player, nil, game)
	player.BroadcastState(g.world.AsWorldLike(), entityID)
	g.creationStep(player, "race")
}

// handleScore (the `stats`/`score` command) shows the player's attributes and
// the bonuses they confer.
func (g *Game) handleScore(player *components.Player, args []string, game *Game) {
	entityID, err := g.getPlayerEntity(player)
	if err != nil {
		player.Broadcast("Unable to read your character.")
		return
	}
	stats := g.getStats(entityID)
	if stats == nil {
		player.Broadcast("You have no stats.")
		return
	}

	player.Broadcast(fmt.Sprintf("== %s -- level %d ==", player.Name, g.casterLevel(entityID)))
	if def, ok := classByKey(stats.Class); ok {
		player.Broadcast(fmt.Sprintf("  %s %s   schools: %s", stats.Race, def.Name, strings.Join(def.Schools, ", ")))
	} else {
		player.Broadcast(fmt.Sprintf("  %s (no class -- 'class <name>' to choose)", stats.Race))
	}
	player.Broadcast(fmt.Sprintf("  %s %3d   melee damage x%.2f", components.StatAbbrev(components.STR), stats.Get(components.STR), stats.MeleeFactor()))
	player.Broadcast(fmt.Sprintf("  %s %3d   endurance cost x%.2f", components.StatAbbrev(components.DEX), stats.Get(components.DEX), stats.CostFactor()))
	player.Broadcast(fmt.Sprintf("  %s %3d   +%d max HP", components.StatAbbrev(components.CON), stats.Get(components.CON), stats.HPBonus()))
	player.Broadcast(fmt.Sprintf("  %s %3d   spell damage x%.2f", components.StatAbbrev(components.INT), stats.Get(components.INT), stats.ArcaneFactor()))
	player.Broadcast(fmt.Sprintf("  %s %3d   healing x%.2f", components.StatAbbrev(components.WIS), stats.Get(components.WIS), stats.HealFactor()))
	if sh := g.playerShift(entityID); sh != nil {
		if form, ok := components.FormFor(sh.Form); ok {
			player.Broadcast(fmt.Sprintf("  form: %s (melee %d-%d, ×STR)", form.Name, form.MinDamage, form.MaxDamage))
		}
	}
	player.Broadcast("Stats rise as you act -- fight, cast, heal, take hits, and roam.")
}

func handleSummon(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Usage: summon <player>")
		return
	}

	targetName := strings.ToLower(strings.Join(args, " "))
	targetPlayer, targetEntityID := game.findOnlinePlayer(targetName)
	if targetPlayer == nil {
		player.Broadcast("No such player online.")
		return
	}
	if targetPlayer == player {
		player.Broadcast("You are already here.")
		return
	}
	if player.Area == nil {
		player.Broadcast("You are nowhere.")
		return
	}

	key := game.clientPersistenceKey(player.Client)
	isAdmin := game.resolveAdminStatus(key)
	player.Lock()
	player.IsAdmin = isAdmin
	player.Unlock()

	if !isAdmin {
		if allowed, reason := game.canCastSummon(player, targetPlayer); !allowed {
			player.Broadcast(reason)
			return
		}
	}

	if targetPlayer.Area != nil && targetPlayer.Area != player.Area {
		targetPlayer.Area.Broadcast(fmt.Sprintf("%s vanishes in a flash of light.", targetPlayer.Name), targetPlayer)
		targetPlayer.Area.RemovePlayer(targetPlayer)
	}

	targetPlayer.Area = player.Area
	player.Area.AddPlayer(targetPlayer)

	player.Broadcast(fmt.Sprintf("You summon %s to your location.", targetPlayer.Name))
	targetPlayer.Broadcast(fmt.Sprintf("You have been summoned by %s.", player.Name))
	player.Area.Broadcast(fmt.Sprintf("%s appears in a flash of light.", targetPlayer.Name), targetPlayer)

	targetPlayer.Look(game.world.AsWorldLike())
	if targetEntityID != "" {
		targetPlayer.BroadcastState(game.world.AsWorldLike(), targetEntityID)
	}
}

func handleRecall(player *components.Player, args []string, game *Game) {
	game.playersMu.RLock()
	playerEntity := game.players[player.Name]
	game.playersMu.RUnlock()

	if player.Area == nil {
		player.Area = game.defaultArea
		game.defaultArea.AddPlayer(player)
		player.Broadcast("You gather your senses and return to the safety of Ravenmoor.")
		player.Look(game.world.AsWorldLike())
		if playerEntity != nil {
			player.BroadcastState(game.world.AsWorldLike(), playerEntity.ID)
		}
		return
	}

	if player.Area == game.defaultArea {
		player.Broadcast("You are already in Ravenmoor.")
		return
	}

	player.Area.RemovePlayer(player)
	player.Area = game.defaultArea
	game.defaultArea.AddPlayer(player)
	player.Broadcast("You focus for a moment and recall to the town of Ravenmoor.\n")
	player.Look(game.world.AsWorldLike())
	if playerEntity != nil {
		player.BroadcastState(game.world.AsWorldLike(), playerEntity.ID)
	}
}

func (g *Game) HandleRename(player *components.Player, newName string) {
	newName = strings.TrimSpace(newName)
	if newName == "" {
		player.Broadcast("Usage: name <new_name>")
		return
	}

	player.Lock()
	oldName := player.Name
	player.Unlock()

	g.playersMu.Lock()
	if _, exists := g.players[newName]; exists {
		g.playersMu.Unlock()
		player.Broadcast(fmt.Sprintf("The name %s is already taken.", newName))
		return
	}
	ent := g.players[oldName]
	delete(g.players, oldName)
	g.players[newName] = ent
	g.playersMu.Unlock()

	player.Lock()
	player.Name = newName
	player.Unlock()

	g.Broadcast(fmt.Sprintf("%s has changed their name to %s", oldName, newName))
	g.creationStep(player, "name")
}

func (g *Game) HandleLogin(player *components.Player, identity string) {
	if player == nil || player.Client == nil {
		return
	}

	identity = strings.TrimSpace(identity)
	if identity == "" {
		player.Broadcast("Usage: login <uuid>")
		return
	}

	if !g.persistenceEnabled() {
		player.Broadcast(g.persistenceDisabledMessage())
		return
	}

	currentKey := g.clientPersistenceKey(player.Client)
	if currentKey == identity {
		player.Broadcast("Already logged in with that id.")
		return
	}

	entityID, err := g.getPlayerEntity(player)
	if err != nil {
		player.Broadcast("Unable to locate your character.")
		return
	}
	entity, err := g.world.FindEntity(entityID)
	if err != nil {
		player.Broadcast("Unable to locate your character.")
		return
	}

	state, err := g.store.LoadPlayer(context.Background(), identity)
	if err != nil {
		player.Broadcast("Failed to load saved profile.")
		return
	}

	if state == nil {
		player.Broadcast("No saved profile found for that id.")
		return
	}

	g.setClientPersistenceKey(player.Client, identity)

	player.Lock()
	player.IsAdmin = g.resolveAdminStatus(identity)
	player.Unlock()

	if state.Name != "" && state.Name != player.Name {
		g.playersMu.Lock()
		if existing, ok := g.players[state.Name]; ok && existing != nil && existing.ID != entity.ID {
			g.playersMu.Unlock()
			player.Broadcast("Saved name is already in use; keeping your current name.")
		} else {
			delete(g.players, player.Name)
			g.players[state.Name] = &entity
			g.playersMu.Unlock()
			player.Lock()
			player.Name = state.Name
			player.Unlock()
		}
	}

	g.applyPlayerState(entity.ID, player, state)
	g.ensureCreation(entity.ID, player) // a loaded-but-unfinished ghost resumes creation

	addedToArea := false
	if area := g.resolveArea(state.AreaID); area != nil && area != player.Area {
		if player.Area != nil {
			player.Area.RemovePlayer(player)
		}
		player.Area = area
		area.AddPlayer(player)
		addedToArea = true
	}

	player.Lock()
	enteredWorld := player.EnteredWorld
	player.Unlock()
	if !enteredWorld {
		if player.Area == nil {
			player.Area = g.defaultArea
		}
		if !addedToArea {
			player.Area.AddPlayer(player)
		}
		player.Lock()
		player.EnteredWorld = true
		player.Unlock()
		g.Broadcast(fmt.Sprintf("%s has joined the game.", player.Name), player.Client)
	}

	if !player.Client.SupportsTags() {
		player.Broadcast("Profile loaded.")
		player.Broadcast("\n")
	}
	player.Look(g.world.AsWorldLike())
	player.BroadcastState(g.world.AsWorldLike(), entity.ID)
	g.announceIdentity(player, identity)
}

func (g *Game) HandleSave(player *components.Player) {
	if player == nil || player.Client == nil {
		return
	}

	if !g.persistenceEnabled() {
		player.Broadcast(g.persistenceDisabledMessage())
		return
	}

	entityID, err := g.getPlayerEntity(player)
	if err != nil {
		player.Broadcast("Unable to locate your character.")
		return
	}
	entity, err := g.world.FindEntity(entityID)
	if err != nil {
		player.Broadcast("Unable to locate your character.")
		return
	}

	key := g.clientPersistenceKey(player.Client)
	if key == "" {
		key = uuid.New().String()
		g.setClientPersistenceKey(player.Client, key)
	}

	player.Lock()
	player.IsAdmin = g.resolveAdminStatus(key)
	player.Unlock()

	if err := g.savePlayerState(key, &entity, player); err != nil {
		player.Broadcast("Save failed.")
		return
	}
	if saved, err := g.store.LoadPlayer(context.Background(), key); err != nil || saved == nil {
		player.Broadcast("Save failed to persist.")
		return
	}
	player.Broadcast("Saved. Your login id: " + key)
	g.announceIdentity(player, key)
}

func handleTime(player *components.Player, args []string, game *Game) {
	if game.dayCycleSystem == nil {
		player.Broadcast("The flow of time seems uncertain here.")
		return
	}

	dc := game.dayCycleSystem.GetDayCycle()
	dc.RLock()
	defer dc.RUnlock()

	remaining := dc.GetCurrentPeriodDuration() - dc.ElapsedTime
	mins := int(remaining.Minutes())
	secs := int(remaining.Seconds()) % 60

	phase := components.MoonPhaseForDay(dc.DayNumber)
	player.Broadcast(fmt.Sprintf("Day %d - It is currently %s. The moon is a %s.", dc.DayNumber, dc.CurrentTime.String(), phase.String()))
	player.Broadcast(dc.GetDescription())
	player.Broadcast(fmt.Sprintf("Time until next period: %d minutes, %d seconds.", mins, secs))
}

func handleExamine(player *components.Player, args []string, game *Game) {
	if len(args) == 0 {
		player.Broadcast("Examine what?")
		return
	}

	target := strings.Join(args, " ")

	// Check if examining self
	if strings.ToLower(target) == "self" || strings.ToLower(target) == "me" || strings.ToLower(target) == player.Name {
		game.playersMu.RLock()
		playerEntity := game.players[player.Name]
		game.playersMu.RUnlock()

		if playerEntity != nil {
			var msg strings.Builder
			msg.WriteString("You examine yourself.\n")

			health, err := game.world.GetComponent(playerEntity.ID, "Health")
			if err == nil {
				h := health.(*components.Health)
				statusEffects, _ := game.world.GetComponent(playerEntity.ID, "StatusEffects")
				bonus := 0
				if statusEffects != nil {
					se := statusEffects.(*components.StatusEffects)
					bonus = se.GetTotalHPBonus()
				}
				effectiveMax := h.GetEffectiveMax(bonus)
				msg.WriteString(fmt.Sprintf("Health: %d/%d HP\n", h.Current, effectiveMax))
			}

			statusEffects, err := game.world.GetComponent(playerEntity.ID, "StatusEffects")
			if err == nil && statusEffects != nil {
				se := statusEffects.(*components.StatusEffects)
				se.RLock()
				if len(se.Effects) > 0 {
					msg.WriteString("\nActive Effects:\n")
					for _, effect := range se.Effects {
						msg.WriteString(fmt.Sprintf("  - %s (+%d HP)\n", effect.Name, effect.HPBonus))
					}
				}
				se.RUnlock()
			}

			if mountComp, err := game.world.GetComponent(playerEntity.ID, "Mount"); err == nil {
				if m, ok := mountComp.(*components.Mount); ok {
					m.Regen()
					msg.WriteString(fmt.Sprintf("\nMounted on a %s (move speed %d, endurance %d/%d).\n", m.Name, m.Speed, m.Endurance, m.MaxEndurance))
				}
			}

			player.Broadcast(msg.String())
			return
		}
	}

	// Check for NPCs in the area
	npcs := player.Area.GetNPCs(game.world.AsWorldLike())
	for _, npc := range npcs {
		if strings.Contains(strings.ToLower(npc.Name), strings.ToLower(target)) {
			player.Broadcast(npc.Description)

			// Show health status
			npcEntities, _ := game.world.FindEntitiesByComponentPredicate("NPC", func(i interface{}) bool {
				n, ok := i.(*components.NPC)
				return ok && n == npc
			})

			if len(npcEntities) > 0 {
				health, err := game.world.GetComponent(npcEntities[0].ID, "Health")
				if err == nil {
					h := health.(*components.Health)
					healthPercent := float64(h.Current) / float64(h.Max) * 100

					var status string
					switch {
					case healthPercent >= 90:
						status = "is in excellent condition"
					case healthPercent >= 70:
						status = "has a few scratches"
					case healthPercent >= 50:
						status = "is wounded"
					case healthPercent >= 30:
						status = "is badly wounded"
					case healthPercent >= 10:
						status = "is near death"
					default:
						status = "is dying"
					}

					player.Broadcast(npc.Name + " " + status + ".")
				}
			}

			return
		}
	}

	// Check for players
	targetPlayer := player.Area.GetPlayer(target)
	if targetPlayer != nil {
		player.Broadcast("You see " + targetPlayer.Name + ", a fellow adventurer.")
		return
	}

	player.Broadcast("You don't see that here.")
}

func (g *Game) canCastSummon(_ *components.Player, _ *components.Player) (bool, string) {
	// TODO: Implement class, mana, cooldown, and location restrictions.
	return false, "You lack the knowledge to cast summon."
}

func (g *Game) findOnlinePlayer(name string) (*components.Player, common.EntityID) {
	needle := strings.ToLower(strings.TrimSpace(name))
	if needle == "" {
		return nil, ""
	}

	g.playersMu.RLock()
	defer g.playersMu.RUnlock()

	if entity := g.players[name]; entity != nil {
		if player, ok := g.playerFromEntity(entity.ID); ok {
			return player, entity.ID
		}
	}

	for playerName, entity := range g.players {
		if strings.Contains(strings.ToLower(playerName), needle) {
			if player, ok := g.playerFromEntity(entity.ID); ok {
				return player, entity.ID
			}
		}
	}

	return nil, ""
}

func (g *Game) playerFromEntity(entityID common.EntityID) (*components.Player, bool) {
	playerComp, err := g.world.GetComponent(entityID, "Player")
	if err != nil {
		return nil, false
	}
	player, ok := playerComp.(*components.Player)
	if !ok || player == nil {
		return nil, false
	}
	return player, true
}

// handleSpawnItem is an admin cheat that drops items straight into your
// inventory — handy for testing quest turn-ins and other item content without
// grinding for drops. Usage: spawnitem <item_id> [qty].
func (g *Game) handleSpawnItem(player *components.Player, args []string, game *Game) {
	key := g.clientPersistenceKey(player.Client)
	isAdmin := g.resolveAdminStatus(key)
	player.Lock()
	player.IsAdmin = isAdmin
	player.Unlock()
	if !isAdmin {
		player.Broadcast("You lack permission.")
		return
	}

	if len(args) == 0 {
		player.Broadcast("Usage: spawnitem <item_id> [qty]   (e.g. spawnitem goblin_ear 10)")
		return
	}
	itemID := strings.ToLower(args[0])
	qty := 1
	if len(args) > 1 {
		if n, err := strconv.Atoi(args[1]); err == nil && n > 0 {
			qty = n
		}
	}

	item := components.CreateItem(itemID, qty)
	if item == nil {
		player.Broadcast(fmt.Sprintf("Unknown item template %q.", itemID))
		return
	}

	entityID, err := g.getPlayerEntity(player)
	if err != nil {
		player.Broadcast("Something went wrong.")
		return
	}
	invComp, err := g.world.GetComponent(entityID, "Inventory")
	if err != nil {
		player.Broadcast("You have no inventory.")
		return
	}
	invComp.(*components.Inventory).AddItem(item)
	player.Broadcast(fmt.Sprintf("[admin] Spawned %s x%d into your inventory.", item.Name, qty))
}

func handleAdminStats(player *components.Player, args []string, game *Game) {
	key := game.clientPersistenceKey(player.Client)
	isAdmin := game.resolveAdminStatus(key)
	player.Lock()
	player.IsAdmin = isAdmin
	player.Unlock()
	if !isAdmin {
		player.Broadcast("You lack permission.")
		return
	}

	entityCount := game.world.EntityCount()
	roomCount, exitCount := game.countRoomsAndExits()
	playerCount := game.countByComponent("Player")
	mobCount := game.countByComponent("NPC")
	corpseCount := game.countByComponent("Corpse")
	groundItems, inventoryItems, corpseItems := countItems(game)

	uptime := time.Since(game.StartTime).Truncate(time.Second)

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	alloc := formatBytes(mem.Alloc)
	heap := formatBytes(mem.HeapAlloc)
	sys := formatBytes(mem.Sys)

	player.Broadcast("== Admin Stats ==")
	player.Broadcast(fmt.Sprintf(
		"Uptime: %s",
		uptime,
	))
	player.Broadcast(fmt.Sprintf(
		"Entities: %d  Players: %d  Mobs: %d  Corpses: %d  Rooms: %d  Exits: %d",
		entityCount, playerCount, mobCount, corpseCount, roomCount, exitCount,
	))
	player.Broadcast(fmt.Sprintf(
		"Items:    ground %d  inventory %d  corpses %d",
		groundItems, inventoryItems, corpseItems,
	))
	player.Broadcast(fmt.Sprintf(
		"Memory:   alloc %s  heap %s  sys %s  gc %d",
		alloc, heap, sys, mem.NumGC,
	))
}

func countItems(game *Game) (groundItems int, inventoryItems int, corpseItems int) {
	areas, err := game.world.FindEntitiesByComponentPredicate("Area", func(i interface{}) bool {
		_, ok := i.(*components.Area)
		return ok
	})
	if err == nil {
		for _, entity := range areas {
			areaComp, err := game.world.GetComponent(entity.ID, "Area")
			if err != nil {
				continue
			}
			area, ok := areaComp.(*components.Area)
			if !ok || area == nil {
				continue
			}
			groundItems += len(area.GetItems())
		}
	}

	inventoryEntities, err := game.world.FindEntitiesByComponentPredicate("Inventory", func(i interface{}) bool {
		_, ok := i.(*components.Inventory)
		return ok
	})
	if err == nil {
		for _, entity := range inventoryEntities {
			invComp, err := game.world.GetComponent(entity.ID, "Inventory")
			if err != nil {
				continue
			}
			inv, ok := invComp.(*components.Inventory)
			if !ok || inv == nil {
				continue
			}
			inv.RLock()
			inventoryItems += len(inv.Items)
			inv.RUnlock()
		}
	}

	corpseEntities, err := game.world.FindEntitiesByComponentPredicate("Corpse", func(i interface{}) bool {
		_, ok := i.(*components.Corpse)
		return ok
	})
	if err == nil {
		for _, entity := range corpseEntities {
			corpseComp, err := game.world.GetComponent(entity.ID, "Corpse")
			if err != nil {
				continue
			}
			corpse, ok := corpseComp.(*components.Corpse)
			if !ok || corpse == nil {
				continue
			}
			corpse.RLock()
			if corpse.Inventory != nil {
				corpseItems += len(corpse.Inventory.Items)
			}
			corpse.RUnlock()
		}
	}

	return groundItems, inventoryItems, corpseItems
}

func formatBytes(value uint64) string {
	const mb = 1024 * 1024
	return fmt.Sprintf("%.1fMB", float64(value)/float64(mb))
}
