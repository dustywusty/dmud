package ecs

import (
	"fmt"
	"reflect"
	"sync"

	"dmud/internal/common"
	"dmud/internal/components"
	"dmud/internal/util"

	"github.com/rs/zerolog/log"
)

type World struct {
	components     map[common.EntityID]map[string]Component
	componentMutex sync.RWMutex

	entities    map[common.EntityID]Entity
	entityMutex sync.RWMutex

	elapsedTime float64

	systems []System
}

func (w *World) AddComponent(entity *Entity, component Component) {
	w.componentMutex.Lock()
	defer w.componentMutex.Unlock()

	w.entityMutex.Lock()
	defer w.entityMutex.Unlock()

	componentName := reflect.TypeOf(component).Elem().Name()
	if _, ok := w.components[entity.ID]; ok {
		w.components[entity.ID][componentName] = component
	} else {
		w.components[entity.ID] = make(map[string]Component)
		w.components[entity.ID][componentName] = component
	}

	entity.Components[componentName] = true

	log.Info().Msgf("Added component %s to entity %s", componentName, entity.ID)
}

func (w *World) AddEntity(entity Entity) {
	w.entityMutex.Lock()
	defer w.entityMutex.Unlock()

	w.componentMutex.Lock()
	defer w.componentMutex.Unlock()

	w.entities[entity.ID] = entity
	w.components[entity.ID] = make(map[string]Component)

	log.Info().Msgf("Added entity %s", entity.ID)
}

func (w *World) AddSystem(system System) {
	w.systems = append(w.systems, system)
}

func (w *World) Components() map[common.EntityID]map[string]Component {
	w.entityMutex.RLock()
	defer w.entityMutex.RUnlock()

	return w.components
}

func (w *World) Entities() map[common.EntityID]Entity {
	w.entityMutex.RLock()
	defer w.entityMutex.RUnlock()

	return w.entities
}

func (w *World) FindEntity(id common.EntityID) (Entity, error) {
	w.entityMutex.RLock()
	defer w.entityMutex.RUnlock()

	entity, ok := w.entities[id]

	if !ok {
		return Entity{}, nil
	}

	return entity, nil
}

func (w *World) FindEntitiesByComponentPredicate(componentType string, predicate func(interface{}) bool) ([]Entity, error) {
	w.componentMutex.RLock()
	defer w.componentMutex.RUnlock()

	w.entityMutex.RLock()
	defer w.entityMutex.RUnlock()

	entities := make([]Entity, 0)
	for entityID, components := range w.components {
		component, ok := components[componentType]
		if ok && predicate(component) {
			if entity, exists := w.entities[entityID]; exists {
				entities = append(entities, entity)
			}
		}
	}

	if len(entities) == 0 {
		return nil, nil
	}

	return entities, nil
}

func (w *World) GetComponent(entityID common.EntityID, componentName string) (interface{}, error) {
	w.componentMutex.RLock()
	defer w.componentMutex.RUnlock()

	if components, ok := w.components[entityID]; ok {
		if component, ok := components[componentName]; ok {
			return component, nil
		}
		return nil, fmt.Errorf("component %s not found for entity %s", componentName, entityID)
	}

	return nil, fmt.Errorf("entity %s not found", entityID)
}

func (w *WorldLikeAdapter) RemoveComponent(entityID common.EntityID, componentType string) error {
	w.World.RemoveComponent(entityID, componentType)
	return nil
}

func (w *WorldLikeAdapter) RemoveEntity(entityID common.EntityID) error {
	w.World.RemoveEntity(entityID)
	return nil
}

func (w *World) RemoveComponent(entityID common.EntityID, componentName string) {
	w.componentMutex.Lock()
	defer w.componentMutex.Unlock()

	if _, ok := w.components[entityID]; ok {
		delete(w.components[entityID], componentName)
		if len(w.components[entityID]) == 0 {
			delete(w.components, entityID)
		}
	} else {
		log.Error().Msgf("Entity %s does not have component %s", entityID, componentName)
	}

	log.Info().Msgf("Removed component %s from entity %s", componentName, entityID)
}

func (w *World) RemoveEntity(entityID common.EntityID) {
	if playerComponent, err := w.GetComponent(entityID, "Player"); err == nil {
		player, ok := playerComponent.(*components.Player)
		if !ok {
			log.Error().Msgf("Error type asserting Player for player %s", entityID)
		} else {
			if player.Area != nil {
				player.Area.RemovePlayer(player)
			}
		}
	}

	w.entityMutex.Lock()
	delete(w.entities, entityID)
	w.entityMutex.Unlock()

	w.componentMutex.Lock()
	delete(w.components, entityID)
	w.componentMutex.Unlock()

	log.Info().Msgf("Removed entity %s", entityID)
}

func (w *World) Update() {
	deltaTime := util.CalculateDeltaTime()

	w.elapsedTime += deltaTime

	if w.elapsedTime >= 0.1 {
		for _, system := range w.systems {
			system.Update(w, deltaTime)
		}

		w.elapsedTime = 0
	}
}

func NewWorld() *World {
	world := &World{
		entities:   make(map[common.EntityID]Entity),
		components: make(map[common.EntityID]map[string]Component),
	}

	areas := loadAreasFromFile("./resources/areas.json")

	for _, area := range areas {
		areaEntity := NewEntity(area.ID)
		areaComponent := &components.Area{
			Region:      area.Region,
			Description: area.Description,
		}
		world.AddEntity(areaEntity)
		world.AddComponent(&areaEntity, areaComponent)
	}

	for _, area := range areas {
		component, err := world.GetComponent(common.EntityID(area.ID), "Area")
		if err != nil {
			log.Error().Err(err).Msgf("Could not get Area for area %s", area.ID)
			continue
		}

		areaComponent, ok := component.(*components.Area)
		if !ok {
			log.Error().Msgf("Error type asserting Area for area %s", area.ID)
			continue
		}

		for direction, areaID := range area.Exits {
			exitAreaUntyped, err := world.GetComponent(common.EntityID(areaID), "Area")
			if err != nil {
				log.Error().Err(err).Msgf("Could not get Area for exit area %s", areaID)
				continue
			}

			exitArea, ok := exitAreaUntyped.(*components.Area)
			if !ok {
				log.Error().Msgf("Error type asserting Area for exit area %s", areaID)
				continue
			}

			areaComponent.Exits = append(areaComponent.Exits, components.Exit{
				Direction: direction,
				AreaID:    areaID,
				Area:      exitArea,
			})
		}
	}

	return world
}

type areaDefinition struct {
	ID          string            `json:"id"`
	Region      string            `json:"region"`
	Description string            `json:"description"`
	Exits       map[string]string `json:"exits"`
}

func loadAreasFromFile(filename string) []areaDefinition {
	var areas []areaDefinition
	if err := util.ParseJSON(filename, &areas); err != nil {
		log.Error().Err(err).Msg("")
	}
	return areas
}

type WorldLikeAdapter struct {
	*World
}

func (w *WorldLikeAdapter) FindEntitiesByComponentPredicate(componentType string, predicate func(interface{}) bool) ([]components.EntityLike, error) {
	entities, err := w.World.FindEntitiesByComponentPredicate(componentType, predicate)
	if err != nil {
		return nil, err
	}

	// Convert []Entity to []components.EntityLike
	result := make([]components.EntityLike, len(entities))
	for i, entity := range entities {
		result[i] = entity
	}
	return result, nil
}

func (w *WorldLikeAdapter) GetComponent(entityID common.EntityID, componentType string) (interface{}, error) {
	return w.World.GetComponent(entityID, componentType)
}

func (w *WorldLikeAdapter) CreateEntity() components.EntityLike {
	entity := NewEntity()
	w.World.AddEntity(entity)
	return entity
}

func (w *WorldLikeAdapter) AddComponentToEntity(entity components.EntityLike, component interface{}) {
	// Convert EntityLike back to *Entity
	e, ok := entity.(Entity)
	if !ok {
		log.Error().Msg("Failed to convert EntityLike to Entity")
		return
	}
	w.World.AddComponent(&e, component)
}

func (w *World) AsWorldLike() components.WorldLike {
	return &WorldLikeAdapter{w}
}
