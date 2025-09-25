package storage

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/ecs"
	"dmud/internal/util"

	"github.com/golang-module/carbon/v2"
	"github.com/rs/zerolog/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const defaultPersistInterval = 5 * time.Second

// Storage provides helpers for persisting world state to PostgreSQL.
type Storage struct {
	db       *gorm.DB
	mu       sync.Mutex
	interval time.Duration
}

// NewFromEnv creates a storage instance using the DMUD_DATABASE_DSN environment variable.
func NewFromEnv() (*Storage, error) {
	dsn := os.Getenv("DMUD_DATABASE_DSN")
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("DMUD_DATABASE_DSN environment variable is required")
	}
	return New(dsn)
}

// New initializes a Storage instance from the provided DSN.
func New(dsn string) (*Storage, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	storage := &Storage{db: db, interval: defaultPersistInterval}

	if err := storage.autoMigrate(); err != nil {
		return nil, err
	}

	return storage, nil
}

func (s *Storage) autoMigrate() error {
	return s.db.AutoMigrate(
		&EntityModel{},
		&AreaModel{},
		&AreaExitModel{},
		&PlayerModel{},
		&HealthModel{},
		&MovementModel{},
		&CombatModel{},
		&NPCModel{},
		&SpawnModel{},
	)
}

// PersistInterval returns the configured persistence interval.
func (s *Storage) PersistInterval() time.Duration {
	if s == nil || s.interval == 0 {
		return defaultPersistInterval
	}
	return s.interval
}

// SetPersistInterval updates the persistence interval.
func (s *Storage) SetPersistInterval(d time.Duration) {
	if s == nil {
		return
	}
	s.interval = d
}

// BootstrapWorld ensures the world is loaded from storage or seeded from resources.
// It returns true when the world was seeded from static definitions (on first run).
func (s *Storage) BootstrapWorld(world *ecs.World) (bool, error) {
	if s == nil {
		return false, errors.New("storage is nil")
	}

	var areaCount int64
	if err := s.db.Model(&AreaModel{}).Count(&areaCount).Error; err != nil {
		return false, fmt.Errorf("count areas: %w", err)
	}

	if areaCount == 0 {
		log.Info().Msg("No areas in database, seeding from resources")
		if err := s.seedFromResources(world); err != nil {
			return false, err
		}
		if err := s.SaveWorld(world); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, s.LoadWorld(world)
}

func (s *Storage) seedFromResources(world *ecs.World) error {
	world.PopulateFromFile("./resources/areas.json")
	return nil
}

// LoadWorld hydrates the ECS world from the persistent store.
func (s *Storage) LoadWorld(world *ecs.World) error {
	if s == nil {
		return errors.New("storage is nil")
	}

	var entityModels []EntityModel
	if err := s.db.Find(&entityModels).Error; err != nil {
		return fmt.Errorf("load entities: %w", err)
	}

	for _, model := range entityModels {
		entity := ecs.Entity{
			ID:         common.EntityID(model.ID),
			Components: make(map[string]bool),
			CreatedAt:  carbon.FromStdTime(model.CreatedAt),
			UpdatedAt:  carbon.FromStdTime(model.UpdatedAt),
		}
		world.AddEntity(entity)
	}

	if err := s.loadAreas(world); err != nil {
		return err
	}

	if err := s.loadSpawns(world); err != nil {
		return err
	}

	if err := s.loadNPCs(world); err != nil {
		return err
	}

	if err := s.loadHealth(world); err != nil {
		return err
	}

	if err := s.loadCombat(world); err != nil {
		return err
	}

	if err := s.loadMovement(world); err != nil {
		return err
	}

	if err := s.loadPlayers(world); err != nil {
		return err
	}

	return nil
}

func (s *Storage) loadAreas(world *ecs.World) error {
	var areas []AreaModel
	if err := s.db.Find(&areas).Error; err != nil {
		return fmt.Errorf("load areas: %w", err)
	}

	for _, model := range areas {
		entity, err := world.FindEntity(common.EntityID(model.EntityID))
		if err != nil {
			return fmt.Errorf("find area entity %s: %w", model.EntityID, err)
		}
		area := &components.Area{
			Region:      model.Region,
			Description: model.Description,
			X:           model.X,
			Y:           model.Y,
			Z:           model.Z,
		}
		world.AddComponent(&entity, area)
	}

	var exits []AreaExitModel
	if err := s.db.Find(&exits).Error; err != nil {
		return fmt.Errorf("load area exits: %w", err)
	}

	for _, exit := range exits {
		component, err := world.GetComponent(common.EntityID(exit.AreaID), "Area")
		if err != nil {
			log.Error().Err(err).Msgf("Could not get Area for exit %s", exit.AreaID)
			continue
		}
		areaComponent, ok := component.(*components.Area)
		if !ok {
			log.Error().Msgf("Invalid area component for %s", exit.AreaID)
			continue
		}
		exitComponent := components.Exit{Direction: exit.Direction, AreaID: exit.TargetAreaID}
		if targetUntyped, err := world.GetComponent(common.EntityID(exit.TargetAreaID), "Area"); err == nil {
			if targetArea, ok := targetUntyped.(*components.Area); ok {
				exitComponent.Area = targetArea
			}
		}
		areaComponent.Exits = append(areaComponent.Exits, exitComponent)
	}

	return nil
}

func (s *Storage) loadPlayers(world *ecs.World) error {
	var players []PlayerModel
	if err := s.db.Find(&players).Error; err != nil {
		return fmt.Errorf("load players: %w", err)
	}

	for _, model := range players {
		entity, err := world.FindEntity(common.EntityID(model.EntityID))
		if err != nil {
			log.Error().Err(err).Msgf("find player entity %s", model.EntityID)
			continue
		}
		areaComponent, err := s.getAreaComponent(world, model.AreaID)
		if err != nil {
			log.Error().Err(err).Msgf("player %s area missing", model.Name)
		}
		playerComponent := &components.Player{
			Name:           model.Name,
			Area:           areaComponent,
			CommandHistory: components.NewCommandHistory(),
			AutoComplete:   util.NewAutoComplete(),
			LastKnownAddr:  model.LastKnownAddress,
		}
		for _, cmd := range model.CommandHistory {
			playerComponent.CommandHistory.AddCommand(cmd)
		}
		world.AddComponent(&entity, playerComponent)
	}

	return nil
}

func (s *Storage) loadHealth(world *ecs.World) error {
	var healthModels []HealthModel
	if err := s.db.Find(&healthModels).Error; err != nil {
		return fmt.Errorf("load health: %w", err)
	}

	for _, model := range healthModels {
		entity, err := world.FindEntity(common.EntityID(model.EntityID))
		if err != nil {
			log.Error().Err(err).Msgf("find health entity %s", model.EntityID)
			continue
		}
		health := &components.Health{Current: model.Current, Max: model.Max, Status: components.HealthStatus(model.Status)}
		world.AddComponent(&entity, health)
	}
	return nil
}

func (s *Storage) loadCombat(world *ecs.World) error {
	var combatModels []CombatModel
	if err := s.db.Find(&combatModels).Error; err != nil {
		return fmt.Errorf("load combat: %w", err)
	}

	for _, model := range combatModels {
		entity, err := world.FindEntity(common.EntityID(model.EntityID))
		if err != nil {
			log.Error().Err(err).Msgf("find combat entity %s", model.EntityID)
			continue
		}
		combat := &components.Combat{TargetID: common.EntityID(model.TargetID), MinDamage: model.MinDamage, MaxDamage: model.MaxDamage}
		world.AddComponent(&entity, combat)
	}
	return nil
}

func (s *Storage) loadMovement(world *ecs.World) error {
	var movementModels []MovementModel
	if err := s.db.Find(&movementModels).Error; err != nil {
		return fmt.Errorf("load movement: %w", err)
	}

	for _, model := range movementModels {
		entity, err := world.FindEntity(common.EntityID(model.EntityID))
		if err != nil {
			log.Error().Err(err).Msgf("find movement entity %s", model.EntityID)
			continue
		}
		movement := &components.Movement{Status: components.MovementStatus(model.Status), Direction: model.Direction}
		world.AddComponent(&entity, movement)
	}
	return nil
}

func (s *Storage) loadNPCs(world *ecs.World) error {
	var npcModels []NPCModel
	if err := s.db.Find(&npcModels).Error; err != nil {
		return fmt.Errorf("load npcs: %w", err)
	}

	for _, model := range npcModels {
		entity, err := world.FindEntity(common.EntityID(model.EntityID))
		if err != nil {
			log.Error().Err(err).Msgf("find npc entity %s", model.EntityID)
			continue
		}
		areaComponent, err := s.getAreaComponent(world, model.AreaID)
		if err != nil {
			log.Error().Err(err).Msgf("npc %s area missing", model.Name)
		}
		npc := &components.NPC{
			Name:         model.Name,
			Description:  model.Description,
			Area:         areaComponent,
			TemplateID:   model.TemplateID,
			Behavior:     components.NPCBehavior(model.Behavior),
			Dialogue:     []string(model.Dialogue),
			LastAction:   model.LastAction,
			LastMovement: model.LastMovement,
			Target:       common.EntityID(model.TargetID),
		}
		world.AddComponent(&entity, npc)
	}

	return nil
}

func (s *Storage) loadSpawns(world *ecs.World) error {
	var spawnModels []SpawnModel
	if err := s.db.Find(&spawnModels).Error; err != nil {
		return fmt.Errorf("load spawns: %w", err)
	}

	for _, model := range spawnModels {
		entity, err := world.FindEntity(common.EntityID(model.EntityID))
		if err != nil {
			log.Error().Err(err).Msgf("find spawn entity %s", model.EntityID)
			continue
		}
		spawn := components.NewSpawn(common.EntityID(model.AreaID))
		spawn.LastSpawn = model.LastSpawn
		spawn.ActiveSpawns = make(map[string]common.EntityID)
		for k, v := range model.ActiveSpawns {
			spawn.ActiveSpawns[k] = common.EntityID(v)
		}
		configs := make([]components.SpawnConfig, 0, len(model.Configs))
		for _, cfg := range model.Configs {
			configs = append(configs, components.SpawnConfig{
				Type:        components.SpawnType(cfg.Type),
				TemplateID:  cfg.TemplateID,
				MinCount:    cfg.MinCount,
				MaxCount:    cfg.MaxCount,
				RespawnTime: time.Duration(cfg.RespawnSeconds) * time.Second,
				Chance:      cfg.Chance,
			})
		}
		spawn.Configs = configs
		world.AddComponent(&entity, spawn)
	}

	return nil
}

func (s *Storage) getAreaComponent(world *ecs.World, areaID string) (*components.Area, error) {
	if areaID == "" {
		return nil, errors.New("area id empty")
	}
	areaUntyped, err := world.GetComponent(common.EntityID(areaID), "Area")
	if err != nil {
		return nil, err
	}
	area, ok := areaUntyped.(*components.Area)
	if !ok {
		return nil, fmt.Errorf("invalid area component %s", areaID)
	}
	return area, nil
}

// SaveWorld persists the current state of the world into the database.
func (s *Storage) SaveWorld(world *ecs.World) error {
	if s == nil {
		return errors.New("storage is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	entities, componentMap := world.Snapshot()

	entityIDs := make([]string, 0, len(entities))
	entityModels := make([]EntityModel, 0, len(entities))

	for id, entity := range entities {
		entityIDs = append(entityIDs, string(id))
		entityModels = append(entityModels, EntityModel{
			ID:        string(id),
			CreatedAt: entity.CreatedAt.ToStdTime(),
			UpdatedAt: time.Now(),
		})
	}

	areaModels := make([]AreaModel, 0)
	exitModels := make([]AreaExitModel, 0)
	playerModels := make([]PlayerModel, 0)
	healthModels := make([]HealthModel, 0)
	movementModels := make([]MovementModel, 0)
	combatModels := make([]CombatModel, 0)
	npcModels := make([]NPCModel, 0)
	spawnModels := make([]SpawnModel, 0)

	for id, comps := range componentMap {
		if comp, exists := comps["Area"]; exists {
			if areaComponent, ok := comp.(*components.Area); ok {
				areaModels = append(areaModels, AreaModel{
					EntityID:    string(id),
					Region:      areaComponent.Region,
					Description: areaComponent.Description,
					X:           areaComponent.X,
					Y:           areaComponent.Y,
					Z:           areaComponent.Z,
				})
				for _, exit := range areaComponent.Exits {
					exitModels = append(exitModels, AreaExitModel{
						AreaID:       string(id),
						Direction:    exit.Direction,
						TargetAreaID: exit.AreaID,
					})
				}
			}
		}
		if comp, exists := comps["Player"]; exists {
			if playerComponent, ok := comp.(*components.Player); ok {
				history := JSONStringSlice{}
				if playerComponent.CommandHistory != nil {
					history = JSONStringSlice(playerComponent.CommandHistory.GetHistory())
				}
				playerModels = append(playerModels, PlayerModel{
					EntityID: string(id),
					Name:     playerComponent.Name,
					AreaID: func() string {
						if playerComponent.Area != nil {
							return s.findAreaIDByPointer(componentMap, playerComponent.Area)
						}
						return ""
					}(),
					CommandHistory:   history,
					LastKnownAddress: playerComponent.LastKnownAddr,
				})
			}
		}
		if comp, exists := comps["Health"]; exists {
			if healthComponent, ok := comp.(*components.Health); ok {
				healthModels = append(healthModels, HealthModel{
					EntityID: string(id),
					Current:  healthComponent.Current,
					Max:      healthComponent.Max,
					Status:   int(healthComponent.Status),
				})
			}
		}
		if comp, exists := comps["Movement"]; exists {
			if movementComponent, ok := comp.(*components.Movement); ok {
				movementModels = append(movementModels, MovementModel{
					EntityID:  string(id),
					Status:    int(movementComponent.Status),
					Direction: movementComponent.Direction,
				})
			}
		}
		if comp, exists := comps["Combat"]; exists {
			if combatComponent, ok := comp.(*components.Combat); ok {
				combatModels = append(combatModels, CombatModel{
					EntityID:  string(id),
					TargetID:  string(combatComponent.TargetID),
					MinDamage: combatComponent.MinDamage,
					MaxDamage: combatComponent.MaxDamage,
				})
			}
		}
		if comp, exists := comps["NPC"]; exists {
			if npcComponent, ok := comp.(*components.NPC); ok {
				areaID := s.findAreaIDByPointer(componentMap, npcComponent.Area)
				npcModels = append(npcModels, NPCModel{
					EntityID:     string(id),
					Name:         npcComponent.Name,
					Description:  npcComponent.Description,
					AreaID:       areaID,
					TemplateID:   npcComponent.TemplateID,
					Behavior:     int(npcComponent.Behavior),
					Dialogue:     JSONStringSlice(npcComponent.Dialogue),
					LastAction:   npcComponent.LastAction,
					LastMovement: npcComponent.LastMovement,
					TargetID:     string(npcComponent.Target),
				})
			}
		}
		if comp, exists := comps["Spawn"]; exists {
			if spawnComponent, ok := comp.(*components.Spawn); ok {
				cfgs := make([]SpawnConfigRecord, 0, len(spawnComponent.Configs))
				for _, cfg := range spawnComponent.Configs {
					cfgs = append(cfgs, SpawnConfigRecord{
						Type:           int(cfg.Type),
						TemplateID:     cfg.TemplateID,
						MinCount:       cfg.MinCount,
						MaxCount:       cfg.MaxCount,
						RespawnSeconds: int64(cfg.RespawnTime / time.Second),
						Chance:         cfg.Chance,
					})
				}
				active := make(map[string]string)
				for k, v := range spawnComponent.ActiveSpawns {
					active[k] = string(v)
				}
				spawnModels = append(spawnModels, SpawnModel{
					EntityID:     string(id),
					AreaID:       string(spawnComponent.AreaID),
					Configs:      cfgs,
					ActiveSpawns: active,
					LastSpawn:    spawnComponent.LastSpawn,
				})
			}
		}
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&entityModels).Error; err != nil {
			return err
		}

		if err := s.replaceAreaData(tx, areaModels, exitModels, entityIDs); err != nil {
			return err
		}

		if err := s.replaceComponentData(tx, &PlayerModel{}, playerModels, entityIDs); err != nil {
			return err
		}
		if err := s.replaceComponentData(tx, &HealthModel{}, healthModels, entityIDs); err != nil {
			return err
		}
		if err := s.replaceComponentData(tx, &MovementModel{}, movementModels, entityIDs); err != nil {
			return err
		}
		if err := s.replaceComponentData(tx, &CombatModel{}, combatModels, entityIDs); err != nil {
			return err
		}
		if err := s.replaceComponentData(tx, &NPCModel{}, npcModels, entityIDs); err != nil {
			return err
		}
		if err := s.replaceComponentData(tx, &SpawnModel{}, spawnModels, entityIDs); err != nil {
			return err
		}

		return nil
	})
}

func (s *Storage) replaceAreaData(tx *gorm.DB, areas []AreaModel, exits []AreaExitModel, entityIDs []string) error {
	if len(entityIDs) == 0 {
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&AreaModel{}).Error; err != nil {
			return err
		}
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&AreaExitModel{}).Error; err != nil {
			return err
		}
	} else {
		if err := tx.Where("entity_id NOT IN ?", entityIDs).Delete(&AreaModel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("area_id NOT IN ?", entityIDs).Delete(&AreaExitModel{}).Error; err != nil {
			return err
		}
	}

	if len(areas) > 0 {
		if err := tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(&areas).Error; err != nil {
			return err
		}
	}

	if len(exits) > 0 {
		if err := tx.Where("area_id IN ?", entityIDs).Delete(&AreaExitModel{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&exits).Error; err != nil {
			return err
		}
	} else if len(entityIDs) > 0 {
		if err := tx.Where("area_id IN ?", entityIDs).Delete(&AreaExitModel{}).Error; err != nil {
			return err
		}
	}

	return nil
}

func (s *Storage) replaceComponentData(tx *gorm.DB, model interface{}, rows interface{}, entityIDs []string) error {
	if len(entityIDs) == 0 {
		return tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(model).Error
	}
	if err := tx.Where("entity_id NOT IN ?", entityIDs).Delete(model).Error; err != nil {
		return err
	}
	value := reflect.ValueOf(rows)
	if value.Kind() == reflect.Slice && value.Len() == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{UpdateAll: true}).Create(rows).Error
}

func (s *Storage) findAreaIDByPointer(componentMap map[common.EntityID]map[string]ecs.Component, target *components.Area) string {
	if target == nil {
		return ""
	}
	for id, comps := range componentMap {
		if areaComponent, ok := comps["Area"].(*components.Area); ok {
			if areaComponent == target {
				return string(id)
			}
		}
	}
	return ""
}

// FindPlayerByAddress finds a player using the stored last known address.
func (s *Storage) FindPlayerByAddress(addr string) (*PlayerModel, error) {
	if s == nil {
		return nil, errors.New("storage is nil")
	}
	if strings.TrimSpace(addr) == "" {
		return nil, nil
	}
	var model PlayerModel
	if err := s.db.Where("last_known_address = ?", addr).Take(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &model, nil
}

// UpdatePlayerAddress persists a player's last known address immediately.
func (s *Storage) UpdatePlayerAddress(entityID, addr string) error {
	if s == nil {
		return nil
	}
	return s.db.Model(&PlayerModel{}).Where("entity_id = ?", entityID).Update("last_known_address", addr).Error
}
