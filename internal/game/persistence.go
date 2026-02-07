package game

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/persistence"

	"github.com/rs/zerolog/log"
)

const playerStateVersion = 1
const worldStateVersion = 1

func (g *Game) loadPlayerState(c common.Client) (*persistence.PlayerState, string) {
	if !g.persistenceEnabled() {
		return nil, ""
	}
	key := g.clientPersistenceKey(c)
	if key == "" {
		return nil, ""
	}

	state, err := g.store.LoadPlayer(context.Background(), key)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load player state")
		return nil, key
	}
	return state, key
}

func (g *Game) savePlayerState(key string, playerEntity *ecs.Entity, player *components.Player) error {
	if !g.persistenceEnabled() || key == "" || playerEntity == nil || player == nil {
		return nil
	}

	state := g.buildPlayerState(playerEntity.ID, player)
	if state == nil {
		return nil
	}

	if err := g.store.SavePlayer(context.Background(), key, state); err != nil {
		log.Warn().Err(err).Msg("Failed to save player state")
		return err
	}
	return nil
}

func (g *Game) loadWorldState() {
	if !g.persistenceEnabled() {
		return
	}
	state, err := g.store.LoadWorld(context.Background())
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load world state")
		return
	}
	if state == nil {
		return
	}
	g.applyWorldState(state)
}

func (g *Game) saveWorldState() {
	if !g.persistenceEnabled() {
		return
	}
	state := g.buildWorldState()
	if state == nil {
		return
	}
	if err := g.store.SaveWorld(context.Background(), state); err != nil {
		log.Warn().Err(err).Msg("Failed to save world state")
	}
}

func (g *Game) autosaveTick() {
	if !g.persistenceEnabled() {
		return
	}

	start := time.Now()
	playersTotal := 0
	playersSaved := 0
	worldSaved := false

	now := time.Now()
	if g.autosaveInterval > 0 && now.Sub(g.lastAutosave) < g.autosaveInterval {
		return
	}
	g.lastAutosave = now

	g.saveWorldState()
	worldSaved = true

	g.playersMu.RLock()
	defer g.playersMu.RUnlock()

	for _, playerEntity := range g.players {
		playersTotal++
		if playerEntity == nil {
			continue
		}
		playerComp, err := g.world.GetComponent(playerEntity.ID, "Player")
		if err != nil {
			continue
		}
		player, ok := playerComp.(*components.Player)
		if !ok || player == nil || player.Client == nil {
			continue
		}
		if !g.isClientLoggedIn(player.Client) {
			continue
		}
		key := g.clientPersistenceKey(player.Client)
		if key == "" {
			continue
		}
		if err := g.savePlayerState(key, playerEntity, player); err == nil {
			playersSaved++
		}
	}

	log.Debug().
		Dur("elapsed", time.Since(start)).
		Int("players_total", playersTotal).
		Int("players_saved", playersSaved).
		Bool("world_saved", worldSaved).
		Msg("autosave tick")
}

func (g *Game) buildPlayerState(entityID common.EntityID, player *components.Player) *persistence.PlayerState {
	if player == nil {
		return nil
	}

	state := &persistence.PlayerState{
		Version:   playerStateVersion,
		Name:      player.Name,
		AreaID:    g.getAreaID(player.Area),
		UpdatedAt: time.Now().UTC(),
	}

	if healthComp, err := g.world.GetComponent(entityID, "Health"); err == nil {
		if health, ok := healthComp.(*components.Health); ok {
			health.RLock()
			state.Health = persistence.HealthState{
				Current: health.Current,
				Max:     health.Max,
				Status:  health.Status,
			}
			health.RUnlock()
		}
	}

	if expComp, err := g.world.GetComponent(entityID, "Experience"); err == nil {
		if exp, ok := expComp.(*components.Experience); ok {
			exp.RLock()
			state.Experience = persistence.ExperienceState{
				Current: exp.Current,
				Level:   exp.Level,
			}
			exp.RUnlock()
		}
	}

	if invComp, err := g.world.GetComponent(entityID, "Inventory"); err == nil {
		if inv, ok := invComp.(*components.Inventory); ok {
			state.Inventory = buildInventoryState(inv)
		}
	}

	if questsComp, err := g.world.GetComponent(entityID, "PlayerQuests"); err == nil {
		if quests, ok := questsComp.(*components.PlayerQuests); ok {
			quests.RLock()
			state.Quests = make(map[string]persistence.QuestStatusRecord, len(quests.Quests))
			for questID, quest := range quests.Quests {
				if quest == nil {
					continue
				}
				state.Quests[questID] = persistence.QuestStatusRecord{Status: quest.Status}
			}
			quests.RUnlock()
		}
	}

	if effectsComp, err := g.world.GetComponent(entityID, "StatusEffects"); err == nil {
		if effects, ok := effectsComp.(*components.StatusEffects); ok {
			effects.RLock()
			state.StatusEffects = make([]persistence.StatusEffectState, 0, len(effects.Effects))
			for _, effect := range effects.Effects {
				state.StatusEffects = append(state.StatusEffects, persistence.StatusEffectState{
					Type:           effect.Type,
					Name:           effect.Name,
					AppliedAtUnix:  effect.AppliedAt.Unix(),
					DurationSecond: int64(effect.Duration.Seconds()),
					HPBonus:        effect.HPBonus,
					Applied:        effect.Applied,
				})
			}
			effects.RUnlock()
		}
	}

	return state
}

func (g *Game) applyPlayerState(entityID common.EntityID, player *components.Player, state *persistence.PlayerState) {
	if state == nil || player == nil {
		return
	}

	if state.Health.Max > 0 {
		if healthComp, err := g.world.GetComponent(entityID, "Health"); err == nil {
			if health, ok := healthComp.(*components.Health); ok {
				health.Lock()
				health.Max = state.Health.Max
				health.Current = state.Health.Current
				health.Status = state.Health.Status
				health.Unlock()
			}
		}
	}

	if state.Experience.Level > 0 {
		if expComp, err := g.world.GetComponent(entityID, "Experience"); err == nil {
			if exp, ok := expComp.(*components.Experience); ok {
				exp.Lock()
				exp.Level = state.Experience.Level
				exp.Current = state.Experience.Current
				exp.Unlock()
			}
		}
	}

	if invComp, err := g.world.GetComponent(entityID, "Inventory"); err == nil {
		if inv, ok := invComp.(*components.Inventory); ok {
			inv.Lock()
			inv.Items = make([]*components.Item, 0, len(state.Inventory.Items))
			inv.MaxSlots = 0
			for _, itemState := range state.Inventory.Items {
				item := itemFromState(itemState)
				if item != nil {
					inv.Items = append(inv.Items, item)
				}
			}
			inv.Unlock()
		}
	}

	if questsComp, err := g.world.GetComponent(entityID, "PlayerQuests"); err == nil {
		if quests, ok := questsComp.(*components.PlayerQuests); ok {
			quests.Lock()
			quests.Quests = make(map[string]*components.PlayerQuest)
			for questID, record := range state.Quests {
				quests.Quests[questID] = &components.PlayerQuest{
					QuestID: questID,
					Status:  record.Status,
				}
			}
			quests.Unlock()
		}
	}

	if len(state.StatusEffects) > 0 {
		effects := components.NewStatusEffects()
		now := time.Now()
		for _, saved := range state.StatusEffects {
			duration := time.Duration(saved.DurationSecond) * time.Second
			appliedAt := time.Unix(saved.AppliedAtUnix, 0)
			if duration > 0 && now.Sub(appliedAt) >= duration {
				continue
			}
			effects.Effects = append(effects.Effects, components.StatusEffect{
				Type:      saved.Type,
				Name:      saved.Name,
				AppliedAt: appliedAt,
				Duration:  duration,
				HPBonus:   saved.HPBonus,
				Applied:   saved.Applied,
			})
		}
		if len(effects.Effects) > 0 {
			entity, err := g.world.FindEntity(entityID)
			if err == nil {
				g.world.AddComponent(&entity, effects)
			}
		}
	}
}

func itemFromState(state persistence.ItemState) *components.Item {
	if state.ID != "" {
		if template := components.CreateItem(state.ID, state.Quantity); template != nil {
			template.Name = state.Name
			template.Description = state.Description
			template.Type = state.Type
			template.Value = state.Value
			template.Stackable = state.Stackable
			template.Quantity = state.Quantity
			return template
		}
	}

	return &components.Item{
		ID:          state.ID,
		Name:        state.Name,
		Description: state.Description,
		Type:        state.Type,
		Value:       state.Value,
		Stackable:   state.Stackable,
		Quantity:    state.Quantity,
	}
}

func buildInventoryState(inv *components.Inventory) persistence.InventoryState {
	if inv == nil {
		return persistence.InventoryState{}
	}

	inv.RLock()
	items := make([]persistence.ItemState, 0, len(inv.Items))
	for _, item := range inv.Items {
		if item == nil {
			continue
		}
		item.RLock()
		items = append(items, persistence.ItemState{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			Type:        item.Type,
			Value:       item.Value,
			Stackable:   item.Stackable,
			Quantity:    item.Quantity,
		})
		item.RUnlock()
	}
	state := persistence.InventoryState{
		MaxSlots: inv.MaxSlots,
		Items:    items,
	}
	inv.RUnlock()
	return state
}

func (g *Game) buildWorldState() *persistence.WorldState {
	areas, err := g.world.FindEntitiesByComponentPredicate("Area", func(i interface{}) bool {
		_, ok := i.(*components.Area)
		return ok
	})
	if err != nil || len(areas) == 0 {
		return nil
	}

	state := &persistence.WorldState{
		Version:   worldStateVersion,
		Areas:     make(map[string]persistence.AreaState),
		NPCs:      make([]persistence.NPCState, 0),
		Corpses:   make([]persistence.CorpseState, 0),
		UpdatedAt: time.Now().UTC(),
	}

	for _, entity := range areas {
		areaComp, err := g.world.GetComponent(entity.ID, "Area")
		if err != nil {
			continue
		}
		area, ok := areaComp.(*components.Area)
		if !ok || area == nil {
			continue
		}
		items := area.GetItems()
		if len(items) == 0 {
			state.Areas[string(entity.ID)] = persistence.AreaState{}
			continue
		}
		itemStates := make([]persistence.ItemState, 0, len(items))
		for _, item := range items {
			if item == nil {
				continue
			}
			itemStates = append(itemStates, persistence.ItemState{
				ID:          item.ID,
				Name:        item.Name,
				Description: item.Description,
				Type:        item.Type,
				Value:       item.Value,
				Stackable:   item.Stackable,
				Quantity:    item.Quantity,
			})
		}
		state.Areas[string(entity.ID)] = persistence.AreaState{Items: itemStates}
	}

	if g.dayCycleSystem != nil {
		dc := g.dayCycleSystem.GetDayCycle()
		if dc != nil {
			dc.RLock()
			state.DayCycle = persistence.DayCycleState{
				CurrentTime:    dc.CurrentTime,
				ElapsedSeconds: int64(dc.ElapsedTime.Seconds()),
				CycleStartUnix: dc.CycleStart.Unix(),
				DayNumber:      dc.DayNumber,
			}
			dc.RUnlock()
		}
	}

	npcEntities, _ := g.world.FindEntitiesByComponentPredicate("NPC", func(i interface{}) bool {
		_, ok := i.(*components.NPC)
		return ok
	})
	for _, entity := range npcEntities {
		npcComp, err := g.world.GetComponent(entity.ID, "NPC")
		if err != nil {
			continue
		}
		npc, ok := npcComp.(*components.NPC)
		if !ok || npc == nil {
			continue
		}
		npc.RLock()
		areaID := g.getAreaID(npc.Area)
		name := npc.Name
		description := npc.Description
		behavior := npc.Behavior
		lastAction := npc.LastAction.Unix()
		lastMovement := npc.LastMovement.Unix()
		templateID := npc.TemplateID
		npc.RUnlock()

		var health persistence.HealthState
		if healthComp, err := g.world.GetComponent(entity.ID, "Health"); err == nil {
			if h, ok := healthComp.(*components.Health); ok {
				h.RLock()
				health = persistence.HealthState{Current: h.Current, Max: h.Max, Status: h.Status}
				h.RUnlock()
			}
		}

		var invState persistence.InventoryState
		if invComp, err := g.world.GetComponent(entity.ID, "Inventory"); err == nil {
			if inv, ok := invComp.(*components.Inventory); ok {
				invState = buildInventoryState(inv)
			}
		}

		state.NPCs = append(state.NPCs, persistence.NPCState{
			TemplateID:       templateID,
			AreaID:           areaID,
			Name:             name,
			Description:      description,
			Behavior:         behavior,
			Health:           health,
			Inventory:        invState,
			LastActionUnix:   lastAction,
			LastMovementUnix: lastMovement,
		})
	}

	corpseEntities, _ := g.world.FindEntitiesByComponentPredicate("Corpse", func(i interface{}) bool {
		_, ok := i.(*components.Corpse)
		return ok
	})
	for _, entity := range corpseEntities {
		corpseComp, err := g.world.GetComponent(entity.ID, "Corpse")
		if err != nil {
			continue
		}
		corpse, ok := corpseComp.(*components.Corpse)
		if !ok || corpse == nil {
			continue
		}
		corpse.RLock()
		areaID := g.getAreaID(corpse.Area)
		victimName := corpse.VictimName
		wasPlayer := corpse.WasPlayer
		timeOfDeath := corpse.TimeOfDeath.Unix()
		decaySeconds := int64(corpse.DecayTime.Seconds())
		lootedAtUnix := int64(0)
		if corpse.LootedAt != nil {
			lootedAtUnix = corpse.LootedAt.Unix()
		}
		inventory := corpse.Inventory
		corpse.RUnlock()

		invState := buildInventoryState(inventory)

		state.Corpses = append(state.Corpses, persistence.CorpseState{
			VictimName:   victimName,
			WasPlayer:    wasPlayer,
			AreaID:       areaID,
			TimeOfDeath:  timeOfDeath,
			DecaySeconds: decaySeconds,
			LootedAtUnix: lootedAtUnix,
			Inventory:    invState,
		})
	}

	return state
}

func (g *Game) applyWorldState(state *persistence.WorldState) {
	if state == nil {
		return
	}

	if g.dayCycleSystem != nil {
		dc := g.dayCycleSystem.GetDayCycle()
		if dc != nil {
			dc.Lock()
			dc.CurrentTime = state.DayCycle.CurrentTime
			dc.ElapsedTime = time.Duration(state.DayCycle.ElapsedSeconds) * time.Second
			if state.DayCycle.CycleStartUnix > 0 {
				dc.CycleStart = time.Unix(state.DayCycle.CycleStartUnix, 0)
			}
			if state.DayCycle.DayNumber > 0 {
				dc.DayNumber = state.DayCycle.DayNumber
			}
			dc.Unlock()
		}
	}

	for areaID, areaState := range state.Areas {
		areaComp, err := g.world.GetComponent(common.EntityID(areaID), "Area")
		if err != nil {
			continue
		}
		area, ok := areaComp.(*components.Area)
		if !ok || area == nil {
			continue
		}

		area.ItemsMutex.Lock()
		area.Items = make([]*components.Item, 0, len(areaState.Items))
		for _, itemState := range areaState.Items {
			item := itemFromState(itemState)
			if item != nil {
				area.Items = append(area.Items, item)
			}
		}
		area.ItemsMutex.Unlock()
	}

	for _, saved := range state.NPCs {
		area := g.resolveArea(saved.AreaID)
		if area == nil {
			continue
		}

		npcEntity := ecs.NewEntity()
		g.world.AddEntity(npcEntity)

		npc := &components.NPC{
			Area:         area,
			TemplateID:   saved.TemplateID,
			Name:         saved.Name,
			Description:  saved.Description,
			Behavior:     saved.Behavior,
			LastAction:   time.Unix(saved.LastActionUnix, 0),
			LastMovement: time.Unix(saved.LastMovementUnix, 0),
		}
		if template, ok := components.NPCTemplates[saved.TemplateID]; ok {
			npc.Dialogue = template.Dialogue
			if npc.Name == "" {
				npc.Name = template.Name
			}
			if npc.Description == "" {
				npc.Description = template.Description
			}
			if npc.Behavior == 0 {
				npc.Behavior = template.Behavior
			}
		}
		if npc.LastAction.IsZero() {
			npc.LastAction = time.Now()
		}
		if npc.LastMovement.IsZero() {
			npc.LastMovement = time.Now()
		}
		g.world.AddComponent(&npcEntity, npc)

		health := &components.Health{Current: saved.Health.Current, Max: saved.Health.Max, Status: saved.Health.Status}
		if health.Max == 0 {
			if template, ok := components.NPCTemplates[saved.TemplateID]; ok {
				health.Current = template.Health
				health.Max = template.Health
				health.Status = components.Healthy
			}
		}
		g.world.AddComponent(&npcEntity, health)

		inv := components.NewInventory(saved.Inventory.MaxSlots)
		for _, itemState := range saved.Inventory.Items {
			item := itemFromState(itemState)
			if item != nil {
				inv.AddItem(item)
			}
		}
		g.world.AddComponent(&npcEntity, inv)

		if template, ok := components.NPCTemplates[saved.TemplateID]; ok {
			if template.Behavior == components.BehaviorAggressive || template.Behavior == components.BehaviorGuard {
				combat := &components.Combat{MinDamage: template.MinDamage, MaxDamage: template.MaxDamage}
				g.world.AddComponent(&npcEntity, combat)
			}
		}
	}

	for _, saved := range state.Corpses {
		area := g.resolveArea(saved.AreaID)
		if area == nil {
			continue
		}
		timeOfDeath := time.Unix(saved.TimeOfDeath, 0)
		decay := time.Duration(saved.DecaySeconds) * time.Second
		var lootedAt *time.Time
		if saved.LootedAtUnix > 0 {
			loot := time.Unix(saved.LootedAtUnix, 0)
			lootedAt = &loot
		}

		corpse := &components.Corpse{
			VictimName:  saved.VictimName,
			WasPlayer:   saved.WasPlayer,
			TimeOfDeath: timeOfDeath,
			DecayTime:   decay,
			Area:        area,
			Inventory:   components.NewInventory(0),
			LootedAt:    lootedAt,
		}
		for _, itemState := range saved.Inventory.Items {
			item := itemFromState(itemState)
			if item != nil {
				corpse.Inventory.AddItem(item)
			}
		}

		if corpse.IsDecayed() {
			continue
		}

		corpseEntity := ecs.NewEntity()
		g.world.AddEntity(corpseEntity)
		g.world.AddComponent(&corpseEntity, corpse)
	}

	g.rebuildSpawnTracking()
}

func (g *Game) resolveArea(areaID string) *components.Area {
	if strings.TrimSpace(areaID) == "" {
		return nil
	}
	areaComp, err := g.world.GetComponent(common.EntityID(areaID), "Area")
	if err != nil {
		return nil
	}
	area, ok := areaComp.(*components.Area)
	if !ok {
		return nil
	}
	return area
}

func (g *Game) getAreaID(area *components.Area) string {
	if area == nil {
		return ""
	}
	entities, err := g.world.FindEntitiesByComponentPredicate("Area", func(i interface{}) bool {
		return i == area
	})
	if err != nil || len(entities) == 0 {
		return ""
	}
	return string(entities[0].ID)
}

func (g *Game) isClientLoggedIn(c common.Client) bool {
	if c == nil {
		return false
	}
	_, ok := g.getSessionKey(c)
	return ok
}

func (g *Game) persistenceEnabled() bool {
	if g.store == nil {
		return false
	}
	if _, ok := g.store.(*persistence.NoopStore); ok {
		return false
	}
	return true
}

func (g *Game) persistenceDisabledMessage() string {
	if g.persistenceErr != nil {
		return fmt.Sprintf("Persistence unavailable: %v", g.persistenceErr)
	}
	return "Persistence is not configured."
}

func (g *Game) rebuildSpawnTracking() {
	spawnEntities, err := g.world.FindEntitiesByComponentPredicate("Spawn", func(i interface{}) bool {
		_, ok := i.(*components.Spawn)
		return ok
	})
	if err != nil || len(spawnEntities) == 0 {
		return
	}

	npcEntities, _ := g.world.FindEntitiesByComponentPredicate("NPC", func(i interface{}) bool {
		_, ok := i.(*components.NPC)
		return ok
	})

	for _, spawnEntity := range spawnEntities {
		spawn, err := ecs.GetTypedComponent[*components.Spawn](g.world, spawnEntity.ID, "Spawn")
		if err != nil {
			continue
		}
		area, err := ecs.GetTypedComponent[*components.Area](g.world, spawnEntity.ID, "Area")
		if err != nil {
			continue
		}
		spawn.Lock()
		spawn.ActiveSpawns = make(map[string][]common.EntityID)
		for _, config := range spawn.Configs {
			spawn.ActiveSpawns[config.TemplateID] = make([]common.EntityID, 0)
		}
		for _, npcEntity := range npcEntities {
			npcComp, err := g.world.GetComponent(npcEntity.ID, "NPC")
			if err != nil {
				continue
			}
			npc, ok := npcComp.(*components.NPC)
			if !ok || npc == nil {
				continue
			}
			npc.RLock()
			matchesArea := npc.Area == area
			templateID := npc.TemplateID
			npc.RUnlock()
			if !matchesArea {
				continue
			}
			spawn.ActiveSpawns[templateID] = append(spawn.ActiveSpawns[templateID], npcEntity.ID)
		}
		spawn.Unlock()
	}
}

func (g *Game) clientPersistenceKey(c common.Client) string {
	if c == nil {
		return ""
	}
	if key, ok := g.getSessionKey(c); ok {
		return key
	}
	return ""
}

func (g *Game) getSessionKey(c common.Client) (string, bool) {
	g.sessionKeysMu.RLock()
	key, ok := g.sessionKeys[c]
	g.sessionKeysMu.RUnlock()
	if !ok || strings.TrimSpace(key) == "" {
		return "", false
	}
	return key, true
}

func (g *Game) setClientPersistenceKey(c common.Client, key string) {
	if c == nil {
		return
	}
	key = strings.TrimSpace(key)
	if key == "" {
		g.clearClientPersistenceKey(c)
		return
	}
	g.sessionKeysMu.Lock()
	g.sessionKeys[c] = key
	g.sessionKeysMu.Unlock()
}

func (g *Game) clearClientPersistenceKey(c common.Client) {
	if c == nil {
		return
	}
	g.sessionKeysMu.Lock()
	delete(g.sessionKeys, c)
	g.sessionKeysMu.Unlock()
}

func extractClientIP(remoteAddr string) string {
	ipAddr := remoteAddr
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		ipAddr = remoteAddr[:idx]
	}
	ipAddr = strings.Trim(ipAddr, "[]")
	return ipAddr
}
